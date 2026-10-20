package mail

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"strings"
	"time"

	"github.com/zhany/ops-go/services/monitor/webhook"
)

// backoff 第 n 次尝试前的等待时间（首次不等待）。声明为变量便于单测缩短等待。
var backoff = func(n int) time.Duration {
	switch n {
	case 2:
		return time.Second
	default:
		return 3 * time.Second
	}
}

// Send 发送邮件告警，失败最多尝试 maxAttempts 次，每次尝试回调 onAttempt。
// 返回最后一次尝试结果；error 非空表示配置或构造类硬性错误（未发起投递）。
func Send(ctx context.Context, cfg *Config, recipients []string, payload *webhook.Payload, maxAttempts int, onAttempt func(webhook.Attempt)) (webhook.Attempt, error) {
	if maxAttempts < 1 {
		maxAttempts = webhook.MaxAttempts
	}
	subject := fmt.Sprintf("【%s】%s - %s", payload.SeverityText, payload.Summary, payload.RuleName)
	if payload.Recovered || payload.Status == "resolved" {
		subject = "【已恢复】" + payload.RuleName
	}
	message, err := buildMessage(cfg, recipients, subject, webhook.RenderText(payload))
	if err != nil {
		return webhook.Attempt{}, err
	}

	var last webhook.Attempt
	for n := 1; n <= maxAttempts; n++ {
		if n > 1 {
			timer := time.NewTimer(backoff(n))
			select {
			case <-ctx.Done():
				timer.Stop()
				return last, ctx.Err()
			case <-timer.C:
			}
		}
		attempt := sendOnce(ctx, cfg, recipients, message)
		attempt.Attempt = n
		last = attempt
		if onAttempt != nil {
			onAttempt(attempt)
		}
		if attempt.Success || permanent(attempt.ErrorMsg) {
			return attempt, nil
		}
		if ctx.Err() != nil {
			return last, ctx.Err()
		}
	}
	return last, fmt.Errorf("邮件发送失败，已重试 %d 次: %s", maxAttempts, last.ErrorMsg)
}

// permanent 认证被拒属于确定性失败，重试无意义
func permanent(errorMsg string) bool {
	return strings.Contains(errorMsg, "认证失败")
}

func sendOnce(ctx context.Context, cfg *Config, recipients []string, message []byte) webhook.Attempt {
	done := make(chan webhook.Attempt, 1)
	go func() {
		done <- attemptSend(cfg, recipients, message)
	}()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return webhook.Attempt{Success: false, ErrorMsg: "邮件发送已取消: " + ctx.Err().Error()}
	case attempt := <-done:
		return attempt
	case <-timer.C:
		return webhook.Attempt{Success: false, ErrorMsg: "邮件发送超时"}
	}
}

const smtpTimeout = 20 * time.Second

// smtpConn 极简 SMTP 会话（单管道、逐条等待响应）。
// 不用 net/smtp 的 Client：其 Cmd 未导出，无法实现部分国产邮箱仅支持的 AUTH LOGIN。
type smtpConn struct {
	conn   net.Conn
	reader *bufio.Reader
}

func attemptSend(cfg *Config, recipients []string, message []byte) webhook.Attempt {
	conn, err := dialConn(cfg)
	if err != nil {
		return webhook.Attempt{ErrorMsg: "连接 SMTP 失败: " + err.Error()}
	}
	defer conn.Close()

	c := &smtpConn{conn: conn, reader: bufio.NewReader(conn)}
	code, text, err := c.readResponse()
	if err != nil || code != 220 {
		return webhook.Attempt{ErrorMsg: fmt.Sprintf("SMTP 问候异常: %d %v", code, err)}
	}

	ehlo := ehloLine(cfg)
	code, text, err = c.cmd("EHLO " + ehlo)
	if err != nil || code != 250 {
		return webhook.Attempt{ErrorMsg: fmt.Sprintf("EHLO 失败: %d %v", code, err)}
	}
	extensions := strings.ToUpper(text)

	if cfg.Security == "starttls" {
		if !strings.Contains(extensions, "STARTTLS") {
			return webhook.Attempt{ErrorMsg: "SMTP 服务器不支持 STARTTLS，请在渠道配置中改用「无加密」"}
		}
		if code, text, err = c.cmd("STARTTLS"); err != nil || code != 220 {
			return webhook.Attempt{ErrorMsg: fmt.Sprintf("STARTTLS 失败: %d %s", code, text)}
		}
		tlsConn := tls.Client(c.conn, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if err = tlsConn.Handshake(); err != nil {
			return webhook.Attempt{ErrorMsg: "STARTTLS 握手失败: " + err.Error()}
		}
		c = &smtpConn{conn: tlsConn, reader: bufio.NewReader(tlsConn)}
		if code, _, err = c.readResponse(); err != nil || code != 220 {
			return webhook.Attempt{ErrorMsg: fmt.Sprintf("TLS 建立后问候异常: %d %v", code, err)}
		}
		if code, text, err = c.cmd("EHLO " + ehlo); err != nil || code != 250 {
			return webhook.Attempt{ErrorMsg: fmt.Sprintf("TLS 后 EHLO 失败: %d %v", code, err)}
		}
		extensions = strings.ToUpper(text)
	}

	if cfg.Username != "" {
		if err = c.authenticate(cfg, extensions); err != nil {
			return webhook.Attempt{ErrorMsg: err.Error()}
		}
	}

	if code, text, err = c.cmd("MAIL FROM:<" + cfg.From + ">"); err != nil || code != 250 {
		return webhook.Attempt{ErrorMsg: fmt.Sprintf("MAIL FROM 失败: %d %s", code, text)}
	}
	for _, to := range recipients {
		if code, text, err = c.cmd("RCPT TO:<" + to + ">"); err != nil || (code != 250 && code != 251) {
			return webhook.Attempt{ErrorMsg: fmt.Sprintf("RCPT TO(%s) 失败: %d %s", to, code, text)}
		}
	}
	code, text, err = c.cmd("DATA")
	if err != nil || code != 354 {
		return webhook.Attempt{ErrorMsg: fmt.Sprintf("DATA 失败: %d %s", code, text)}
	}
	if err = c.writeMessage(message); err != nil {
		return webhook.Attempt{ErrorMsg: "邮件内容写入失败: " + err.Error()}
	}
	code, text, err = c.readResponse()
	if err != nil || code != 250 {
		return webhook.Attempt{ErrorMsg: fmt.Sprintf("邮件提交失败: %d %s", code, text)}
	}

	_, _, _ = c.cmd("QUIT")
	return webhook.Attempt{StatusCode: 250, Success: true, Body: "已投递至 " + cfg.Addr()}
}

// cmd 发送一条命令并读取响应
func (c *smtpConn) cmd(line string) (code int, text string, err error) {
	if _, err = c.conn.Write([]byte(line + "\r\n")); err != nil {
		return 0, "", err
	}
	return c.readResponse()
}

// readResponse 读取 SMTP 多行响应，返回状态码与合并文本
func (c *smtpConn) readResponse() (int, string, error) {
	code := 0
	var parts []string
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if len(parts) > 0 {
				return code, strings.Join(parts, "\n"), nil
			}
			return code, "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if len(line) < 3 {
			continue
		}
		if _, convErr := fmt.Sscanf(line[:3], "%d", &code); convErr != nil {
			return code, strings.Join(parts, "\n"), errors.New("SMTP 响应格式异常: " + line)
		}
		body := ""
		if len(line) > 4 {
			body = line[4:]
		}
		parts = append(parts, body)
		if len(line) == 3 || line[3] == ' ' {
			break
		}
	}
	return code, strings.Join(parts, "\n"), nil
}

// writeMessage 按点转义规则写入报文体并以 <CRLF>.<CRLF> 结束
func (c *smtpConn) writeMessage(message []byte) error {
	var b strings.Builder
	lines := strings.Split(strings.ReplaceAll(string(message), "\r\n", "\n"), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, ".") {
			line = "." + line
		}
		b.WriteString(line + "\r\n")
	}
	b.WriteString(".\r\n")
	_, err := io.WriteString(c.conn, b.String())
	return err
}

// authenticate 按配置选择 AUTH PLAIN 或 AUTH LOGIN；服务器明示支持 LOGIN 时优先 LOGIN
func (c *smtpConn) authenticate(cfg *Config, extensions string) error {
	if cfg.Auth == "login" || strings.Contains(extensions, "AUTH LOGIN") && !strings.Contains(extensions, "PLAIN") {
		return c.authLogin(cfg)
	}
	return c.authPlain(cfg)
}

func (c *smtpConn) authPlain(cfg *Config) error {
	payload := base64.StdEncoding.EncodeToString([]byte("\x00" + cfg.Username + "\x00" + cfg.Password))
	code, text, err := c.cmd("AUTH PLAIN " + payload)
	if err != nil {
		return err
	}
	if code != 235 {
		return fmt.Errorf("SMTP 认证失败: %d %s", code, text)
	}
	return nil
}

func (c *smtpConn) authLogin(cfg *Config) error {
	code, text, err := c.cmd("AUTH LOGIN")
	if err != nil {
		return err
	}
	if code != 334 {
		return fmt.Errorf("SMTP 认证失败(AUTH LOGIN): %d %s", code, text)
	}
	code, text, err = c.cmd(base64.StdEncoding.EncodeToString([]byte(cfg.Username)))
	if err != nil {
		return err
	}
	if code != 334 {
		return fmt.Errorf("SMTP 认证失败(用户名被拒): %d %s", code, text)
	}
	code, text, err = c.cmd(base64.StdEncoding.EncodeToString([]byte(cfg.Password)))
	if err != nil {
		return err
	}
	if code != 235 {
		return fmt.Errorf("SMTP 认证失败(密码被拒): %d %s", code, text)
	}
	return nil
}

func dialConn(cfg *Config) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.Dial("tcp", cfg.Addr())
	if err != nil {
		return nil, err
	}
	if cfg.Security == "ssl" {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if err = tlsConn.Handshake(); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return tlsConn, nil
	}
	return conn, nil
}

// ehloLine EHLO 参数，优先取发件人域名
func ehloLine(cfg *Config) string {
	if at := strings.LastIndex(cfg.From, "@"); at >= 0 && at+1 < len(cfg.From) {
		return cfg.From[at+1:]
	}
	return "ops-go.local"
}

// buildMessage 组装 RFC5322 报文（主题按 RFC2047 编码中文，正文 8bit UTF-8）
func buildMessage(cfg *Config, recipients []string, subject, body string) ([]byte, error) {
	encodedSubject := mime.BEncoding.Encode("UTF-8", subject)
	toHeader := strings.Join(recipients, ", ")
	var b strings.Builder
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"))
	fmt.Fprintf(&b, "Message-ID: <ops-go.%d@%s>\r\n", time.Now().UnixNano(), cfg.Host)
	fmt.Fprintf(&b, "From: %s\r\n", cfg.From)
	fmt.Fprintf(&b, "To: %s\r\n", toHeader)
	fmt.Fprintf(&b, "Subject: %s\r\n", encodedSubject)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return []byte(b.String()), nil
}
