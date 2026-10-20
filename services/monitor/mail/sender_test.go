package mail

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zhany/ops-go/services/monitor/webhook"
)

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func init() {
	backoff = func(int) time.Duration { return 0 }
}

func TestParseConfigDefaults(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{"host": "smtp.example.com", "port": 465, "username": "u", "password": "p", "from": "ops@example.com"})
	cfg, err := ParseConfig(string(raw))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if cfg.Security != "ssl" || cfg.Auth != "plain" {
		t.Fatalf("465 端口应默认 ssl+plain: %+v", cfg)
	}
	raw, _ = json.Marshal(map[string]interface{}{"host": "h", "port": 587, "from": "a@b.com"})
	cfg, err = ParseConfig(string(raw))
	if err != nil || cfg.Security != "starttls" {
		t.Fatalf("587 无加密方式应推断 starttls: %+v %v", cfg, err)
	}
}

func TestParseConfigRejects(t *testing.T) {
	cases := []string{
		`{}`,
		`{"host":"h","port":70000,"from":"a@b.com"}`,
		`{"host":"h","port":25,"from":"not-an-email"}`,
		`{"host":"h","port":25,"from":"a@b.com","security":"tls"}`,
		`{"host":"h","port":25,"from":"a@b.com","username":"u"}`,
	}
	for _, raw := range cases {
		if _, err := ParseConfig(raw); err == nil {
			t.Fatalf("非法配置应报错: %s", raw)
		}
	}
}

func TestSplitRecipients(t *testing.T) {
	got, err := SplitRecipients("a@x.com； b@x.com, c@x.com;d@x.com ")
	if err != nil || len(got) != 4 {
		t.Fatalf("混合分隔符解析失败: %v %v", got, err)
	}
	if _, err = SplitRecipients("a@x.com, bad_addr"); err == nil {
		t.Fatal("非法地址应报错")
	}
	if _, err = SplitRecipients(""); err == nil {
		t.Fatal("空收件人应报错")
	}
}

func TestMaskRestorePassword(t *testing.T) {
	original := `{"host":"h","port":465,"from":"a@b.com","username":"u","password":"secret"}`
	masked := MaskPassword(original, "******")
	cfg, err := ParseConfig(masked)
	if err != nil || cfg.Password != "******" {
		t.Fatalf("掩码后应仍可解析: %s %v", masked, err)
	}
	// 提交掩码 → 沿用原密码
	restored := RestorePassword(masked, original, "******")
	cfg, err = ParseConfig(restored)
	if err != nil || cfg.Password != "secret" {
		t.Fatalf("掩码提交应还原原密码: %s %v", restored, err)
	}
	// 提交新密码 → 保留新值
	newSubmit := strings.Replace(masked, "******", "newpw", 1)
	restored = RestorePassword(newSubmit, original, "******")
	if !strings.Contains(restored, "newpw") {
		t.Fatalf("新密码不应被覆盖: %s", restored)
	}
	// 提交空密码 → 沿用原值
	emptySubmit := strings.Replace(masked, `"******"`, `""`, 1)
	restored = RestorePassword(emptySubmit, original, "******")
	if !strings.Contains(restored, "secret") {
		t.Fatalf("空密码应沿用原值: %s", restored)
	}
}

// fakeSMTP 极简假 SMTP 服务器（明文，AUTH PLAIN），返回监听端口与收到内容
type fakeSMTP struct {
	ln       net.Listener
	authOK   bool
	gotMail  string
	gotRcpt  []string
	gotData  string
	greetExt string
}

func startFakeSMTP(t *testing.T, authOK bool) *fakeSMTP {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{ln: ln, authOK: authOK}
	go f.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return f
}

func (f *fakeSMTP) serve() {
	conn, err := f.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	write := func(s string) { _, _ = conn.Write([]byte(s)) }

	write("220 fake smtp\r\n")
	authed := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		up := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(up, "EHLO"):
			write("250-fake\r\n250-AUTH PLAIN LOGIN\r\n250 HELP\r\n")
		case strings.HasPrefix(up, "AUTH PLAIN"):
			if f.authOK && len(line) > 11 {
				if payload, decErr := base64.StdEncoding.DecodeString(line[11:]); decErr == nil && strings.Count(string(payload), "\x00") == 2 {
					authed = true
				}
			}
			if authed {
				write("235 ok\r\n")
			} else {
				write("535 auth failed\r\n")
			}
		case strings.HasPrefix(up, "MAIL FROM"):
			if f.authOK && !authed {
				write("503 need auth\r\n")
				continue
			}
			f.gotMail = line
			write("250 ok\r\n")
		case strings.HasPrefix(up, "RCPT TO"):
			f.gotRcpt = append(f.gotRcpt, line)
			write("250 ok\r\n")
		case up == "DATA":
			write("354 go ahead\r\n")
			var body []string
			for {
				l, err2 := r.ReadString('\n')
				if err2 != nil {
					return
				}
				l = strings.TrimRight(l, "\r\n")
				if l == "." {
					break
				}
				body = append(body, l)
			}
			f.gotData = strings.Join(body, "\n")
			write("250 queued\r\n")
		case up == "QUIT":
			write("221 bye\r\n")
			return
		default:
			write("250 ok\r\n")
		}
	}
}

func testPayload() *webhook.Payload {
	return &webhook.Payload{SeverityText: "警告", Summary: "磁盘告警", RuleName: "磁盘使用率", Status: "firing", Instance: "10.0.0.8"}
}

func TestSendSuccess(t *testing.T) {
	f := startFakeSMTP(t, true)
	port := strings.TrimPrefix(f.ln.Addr().String(), "127.0.0.1:")
	cfg := &Config{Host: "127.0.0.1", Port: atoi(port), Username: "u", Password: "p", From: "ops@example.com", Auth: "plain", Security: "none"}
	attempt, err := Send(context.Background(), cfg, []string{"a@x.com", "b@x.com"}, testPayload(), 1, nil)
	if err != nil || !attempt.Success {
		t.Fatalf("发送应成功: %+v %v", attempt, err)
	}
	if len(f.gotRcpt) != 2 {
		t.Fatalf("收件人应 2 个: %v", f.gotRcpt)
	}
	if !strings.Contains(f.gotData, "Subject: =?UTF-8?") || !strings.Contains(f.gotData, "磁盘使用率") {
		t.Fatalf("正文/主题缺失: %q", f.gotData)
	}
}

func TestSendAuthFailureNoRetry(t *testing.T) {
	f := startFakeSMTP(t, false)
	port := strings.TrimPrefix(f.ln.Addr().String(), "127.0.0.1:")
	cfg := &Config{Host: "127.0.0.1", Port: atoi(port), Username: "u", Password: "bad", From: "ops@example.com", Auth: "plain", Security: "none"}
	var attempts int
	last, err := Send(context.Background(), cfg, []string{"a@x.com"}, testPayload(), 3, func(webhook.Attempt) { attempts++ })
	if err != nil {
		t.Fatalf("认证失败不应返回硬错误: %v", err)
	}
	if last.Success || !strings.Contains(last.ErrorMsg, "认证失败") {
		t.Fatalf("应报告认证失败: %+v", last)
	}
	if attempts != 1 {
		t.Fatalf("认证失败应停止重试, got %d", attempts)
	}
}
