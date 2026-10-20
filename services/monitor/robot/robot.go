// Package robot 实现钉钉/企业微信/飞书群机器人的告警消息构造、签名与出站投递。
// 每家机器人均以「HTTP 200 + 响应体业务码」表示成败，因此发送前校验官方域名，
// 每次重试都会重新构造请求（钉钉/飞书签名带时间戳，过期签名会被拒）。
package robot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zhany/ops-go/services/monitor/webhook"
)

// Kind 机器人平台
type Kind int8

const (
	DingTalk Kind = 3
	WeCom    Kind = 4
	Feishu   Kind = 5
)

// FromChannelType 渠道类型转机器人平台，非机器人类型返回 ok=false
func FromChannelType(t int8) (Kind, bool) {
	switch Kind(t) {
	case DingTalk, WeCom, Feishu:
		return Kind(t), true
	default:
		return 0, false
	}
}

// 官方机器人域名白名单，防止渠道地址被配置为内网地址造成 SSRF
var allowedHostSuffixes = map[Kind][]string{
	DingTalk: {"oapi.dingtalk.com"},
	WeCom:    {"qyapi.weixin.qq.com"},
	Feishu:   {"open.feishu.cn"},
}

// ValidateUrl 校验机器人 webhook 地址：必须为 https 且主机属于官方域名
func ValidateUrl(kind Kind, raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fmt.Errorf("机器人地址不能为空")
	}
	if len(trimmed) > 512 {
		return fmt.Errorf("机器人地址过长")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("机器人地址格式非法: %v", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("机器人地址仅支持 https")
	}
	if !hostAllowed(kind, trimmed) {
		return fmt.Errorf("机器人地址必须为官方域名地址")
	}
	return nil
}

// hostAllowed 判断地址主机名是否属于指定平台的官方域名（含子域）
func hostAllowed(kind Kind, raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	for _, suffix := range allowedHostSuffixes[kind] {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

// buildMarkdownBody 构造 {msgtype,markdown:{title,text}} 形式请求体（钉钉/企微结构一致）
func buildMarkdownBody(title, text string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  text,
		},
	})
}

// buildFeishuBody 构造飞书交互卡片请求体
func buildFeishuBody(title, text string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]interface{}{"tag": "plain_text", "content": title},
			},
			"elements": []interface{}{
				map[string]interface{}{"tag": "div", "text": map[string]interface{}{"tag": "lark_md", "content": text}},
			},
		},
	})
}

// RenderMarkdown 渲染机器人通用 markdown 正文
func RenderMarkdown(p *webhook.Payload) string {
	var lines []string
	lines = append(lines, "**规则**: "+p.RuleName)
	lines = append(lines, "**主机**: "+p.Instance)
	if p.Metric != "" && p.Metric != "-" {
		lines = append(lines, fmt.Sprintf("**指标**: %s 当前值: %g 阈值: %g", p.Metric, p.Value, p.Threshold))
	}
	if p.Summary != "" {
		lines = append(lines, "**摘要**: "+p.Summary)
	}
	if p.FiredAt != "" {
		lines = append(lines, "**时间**: "+p.FiredAt)
	}
	return strings.Join(lines, "\n\n")
}

// dingSignURL 按钉钉加签规则在 URL 上追加 timestamp 与 sign
func dingSignURL(raw, secret string, now time.Time) (string, error) {
	ts := strconv.FormatInt(now.UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(ts + "\n" + secret)); err != nil {
		return "", err
	}
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	sep := "?"
	if strings.Contains(raw, "?") {
		sep = "&"
	}
	return raw + sep + "timestamp=" + ts + "&sign=" + sign, nil
}

// feishuSign 按飞书自定义机器人签名算法计算签名头
func feishuSign(secret, body string, now time.Time) (timestamp, nonce, signature string, err error) {
	timestamp = strconv.FormatInt(now.Unix(), 10)
	var rnd [4]byte
	if _, err = rand.Read(rnd[:]); err != nil {
		return "", "", "", err
	}
	nonce = hex.EncodeToString(rnd[:])
	keyMac := hmac.New(sha256.New, nil)
	keyMac.Write([]byte(secret))
	if _, err = keyMac.Write([]byte(timestamp + "\n" + nonce)); err != nil {
		return "", "", "", err
	}
	signingKey := keyMac.Sum(nil)
	mac := hmac.New(sha256.New, signingKey)
	if _, err = mac.Write([]byte(body)); err != nil {
		return "", "", "", err
	}
	return timestamp, nonce, hex.EncodeToString(mac.Sum(nil)), nil
}

// Build 按平台构造一次投递请求；secret 为空则不签名
func Build(kind Kind, p *webhook.Payload, rawUrl, secret string) (*webhook.Request, error) {
	if _, ok := FromChannelType(int8(kind)); !ok {
		return nil, fmt.Errorf("未知机器人类型: %d", kind)
	}
	if !checkHost(kind, rawUrl) {
		return nil, fmt.Errorf("机器人地址必须为官方域名地址")
	}
	status := "告警中"
	if p.Recovered || p.Status == "resolved" {
		status = "已恢复"
	}
	title := fmt.Sprintf("【%s】%s %s", p.SeverityText, status, p.RuleName)
	text := RenderMarkdown(p)

	var (
		body []byte
		err  error
	)
	req := &webhook.Request{Headers: map[string]string{}, EventId: p.EventId}
	url := strings.TrimSpace(rawUrl)
	switch kind {
	case DingTalk:
		body, err = buildMarkdownBody(title, text)
		if secret != "" {
			url, err = dingSignURL(url, secret, time.Now())
		}
	case WeCom:
		body, err = buildMarkdownBody(title, text)
	case Feishu:
		body, err = buildFeishuBody(title, text)
		if secret != "" {
			var timestamp, nonce, sign string
			timestamp, nonce, sign, err = feishuSign(secret, string(body), time.Now())
			if err == nil {
				req.Headers["X-Lark-Request-Timestamp"] = timestamp
				req.Headers["X-Lark-Request-Nonce"] = nonce
				req.Headers["X-Lark-Signature"] = sign
			}
		}
	}
	if err != nil {
		return nil, err
	}
	req.Url = url
	req.Body = body
	return req, nil
}

// businessError 解析响应体业务码；HTTP 非 2xx 或业务码非 0 均视为失败
func businessError(kind Kind, statusCode int, respBody []byte) error {
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("机器人返回状态码 %d", statusCode)
	}
	var resp struct {
		Code    int    `json:"code"`
		ErrCode int    `json:"errcode"`
		Msg     string `json:"msg"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("无法解析机器人响应: %s", truncate(string(respBody)))
	}
	code := resp.ErrCode
	message := resp.ErrMsg
	if kind == Feishu {
		code = resp.Code
		message = resp.Msg
	}
	if code != 0 {
		return fmt.Errorf("机器人返回错误 %d %s", code, message)
	}
	return nil
}

// backoff 第 n 次尝试前的等待时间（首次不等待）。声明为变量便于单测缩短等待。
var backoff = func(n int) time.Duration {
	switch n {
	case 2:
		return time.Second
	default:
		return 3 * time.Second
	}
}

// client/checkHost 为测试注入点：单测用自签 TLS 假机器人验证全流程，生产固定官方域名白名单
var client = &http.Client{
	Timeout: 10 * time.Second,
	// 机器人地址固定官方域名且无正常跳转场景，禁止重定向以防被引导至内网地址
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

var checkHost = hostAllowed

// Dispatch 发送机器人消息，失败最多尝试 maxAttempts 次（每次重新签名），每次尝试回调 onAttempt。
func Dispatch(ctx context.Context, kind Kind, p *webhook.Payload, rawUrl, secret string, maxAttempts int, onAttempt func(webhook.Attempt)) (webhook.Attempt, error) {
	if maxAttempts < 1 {
		maxAttempts = webhook.MaxAttempts
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
		attempt, err := sendOnce(ctx, kind, p, rawUrl, secret)
		attempt.Attempt = n
		last = attempt
		if onAttempt != nil {
			onAttempt(attempt)
		}
		if err != nil {
			return attempt, err
		}
		if attempt.Success {
			return attempt, nil
		}
		if ctx.Err() != nil {
			return last, ctx.Err()
		}
	}
	return last, fmt.Errorf("通知发送失败，已重试 %d 次: %s", maxAttempts, last.ErrorMsg)
}

func sendOnce(ctx context.Context, kind Kind, p *webhook.Payload, rawUrl, secret string) (webhook.Attempt, error) {
	req, err := Build(kind, p, rawUrl, secret)
	if err != nil {
		return webhook.Attempt{ErrorMsg: err.Error()}, nil
	}
	httpCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, req.Url, bytes.NewReader(req.Body))
	if err != nil {
		return webhook.Attempt{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	httpReq.Header.Set("User-Agent", "ops-go-monitor/1.0")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return webhook.Attempt{Success: false, ErrorMsg: err.Error()}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	attempt := webhook.Attempt{StatusCode: resp.StatusCode, Body: string(body)}
	if bizErr := businessError(kind, resp.StatusCode, body); bizErr != nil {
		attempt.ErrorMsg = bizErr.Error()
		return attempt, nil
	}
	attempt.Success = true
	return attempt, nil
}

func truncate(s string) string {
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
