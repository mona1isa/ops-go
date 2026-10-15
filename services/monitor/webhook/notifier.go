// Package webhook 实现告警通知的出站投递：构造请求体、HMAC 签名、重试。
// 该包不依赖数据库，便于独立验证发送逻辑；落库由上层服务通过回调完成。
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// SignatureHeader 签名头，值为 sha256=<hex>
	SignatureHeader = "X-Ops-Signature"
	// EventHeader 告警事件ID头，便于接收端幂等
	EventHeader = "X-Ops-Event-Id"
	// MaxAttempts 单次通知最多尝试次数
	MaxAttempts = 3
)

// Attempt 一次投递的结果，供上层写通知日志
type Attempt struct {
	Attempt    int
	StatusCode int
	Body       string
	Success    bool
	ErrorMsg   string
}

// Request 一次通知投递所需的全部信息
type Request struct {
	Url     string
	Secret  string
	Headers map[string]string
	Body    []byte
	EventId uint64
}

// Payload 告警通知内容
type Payload struct {
	EventId      uint64            `json:"eventId"`
	RuleId       int               `json:"ruleId"`
	RuleName     string            `json:"ruleName"`
	Severity     int8              `json:"severity"`
	SeverityText string            `json:"severityText"`
	Status       string            `json:"status"`
	Recovered    bool              `json:"recovered"`
	Instance     string            `json:"instance"`
	Metric       string            `json:"metric"`
	Value        float64           `json:"value"`
	Threshold    float64           `json:"threshold"`
	Summary      string            `json:"summary"`
	FiredAt      string            `json:"firedAt"`
	Labels       map[string]string `json:"labels"`
}

// SeverityText 级别数值转中文描述
func SeverityText(severity int8) string {
	switch severity {
	case 1:
		return "紧急"
	case 3:
		return "提醒"
	default:
		return "警告"
	}
}

// BuildBody 序列化通知内容
func BuildBody(payload *Payload) ([]byte, error) {
	return json.Marshal(payload)
}

// Sign 计算 HMAC-SHA256 签名，返回 sha256=<hex> 形式
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify 校验签名（供接收端与单测使用）
func Verify(secret string, body []byte, header string) bool {
	expected := Sign(secret, body)
	return hmac.Equal([]byte(expected), []byte(header))
}

func timeout() time.Duration {
	sec := 10
	if raw := os.Getenv("WEBHOOK_TIMEOUT"); raw != "" {
		if num, err := strconv.Atoi(raw); err == nil && num > 0 {
			sec = num
		}
	}
	return time.Duration(sec) * time.Second
}

var client = newClient()

func newClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          16,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// backoff 第 n 次尝试前的等待时间（n 从 1 开始，首次不等待）。声明为变量便于单测缩短等待。
var backoff = func(n int) time.Duration {
	switch n {
	case 2:
		return time.Second
	default:
		return 3 * time.Second
	}
}

// sendOnce 单次投递，返回 attempt 结果
func sendOnce(ctx context.Context, req *http.Request) Attempt {
	result := Attempt{}
	resp, err := client.Do(req)
	if err != nil {
		result.ErrorMsg = err.Error()
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	result.StatusCode = resp.StatusCode
	result.Body = string(body)
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	if !result.Success {
		result.ErrorMsg = fmt.Sprintf("通知渠道返回状态码 %d", resp.StatusCode)
	}
	return result
}

// Dispatch 按退避策略重试投递，每次尝试都会回调 onAttempt（可能为 nil）。
// 返回最后一次尝试的结果；只要有一次成功就立即停止重试。
func Dispatch(ctx context.Context, r *Request, onAttempt func(Attempt)) (Attempt, error) {
	if err := validate(r.Url); err != nil {
		return Attempt{}, err
	}
	var last Attempt
	for n := 1; n <= MaxAttempts; n++ {
		if n > 1 {
			timer := time.NewTimer(backoff(n))
			select {
			case <-ctx.Done():
				timer.Stop()
				return last, ctx.Err()
			case <-timer.C:
			}
		}

		attemptCtx, cancel := context.WithTimeout(ctx, timeout())
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, r.Url, bytes.NewReader(r.Body))
		if err != nil {
			cancel()
			return last, fmt.Errorf("构造通知请求失败: %w", err)
		}
		applyHeaders(req, r)
		last = sendOnce(attemptCtx, req)
		cancel()

		last.Attempt = n
		if onAttempt != nil {
			onAttempt(last)
		}
		if last.Success {
			return last, nil
		}
		if ctx.Err() != nil {
			return last, ctx.Err()
		}
	}
	return last, fmt.Errorf("通知发送失败，已重试 %d 次: %s", MaxAttempts, last.ErrorMsg)
}

// applyHeaders 设置请求头。自定义头不允许覆盖签名与 Content-Type 之外的保留头。
func applyHeaders(req *http.Request, r *Request) {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", "ops-go-monitor/1.0")
	for key, value := range r.Headers {
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(value) == "" {
			continue
		}
		switch strings.ToLower(key) {
		case "content-type", "host", "content-length", "transfer-encoding", "x-ops-signature", "x-ops-event-id":
			continue
		}
		req.Header.Set(key, value)
	}
	if r.Secret != "" {
		req.Header.Set(SignatureHeader, Sign(r.Secret, r.Body))
	}
	if r.EventId != 0 {
		req.Header.Set(EventHeader, strconv.FormatUint(r.EventId, 10))
	}
}

// validate 校验通知地址：渠道地址由管理员配置，此处只保证是可外呼的 http/https 绝对地址
func validate(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("通知地址为空")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("通知地址格式非法: %v", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("通知地址仅支持 http/https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("通知地址缺少主机名")
	}
	return nil
}
