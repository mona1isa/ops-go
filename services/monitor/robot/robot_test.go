package robot

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/zhany/ops-go/services/monitor/webhook"
)

func init() {
	backoff = func(int) time.Duration { return 0 }
}

func samplePayload() *webhook.Payload {
	return &webhook.Payload{
		EventId: 7, RuleId: 1, RuleName: "CPU 使用率过高", Severity: 3, SeverityText: "警告",
		Status: "firing", Instance: "10.0.0.8", Metric: "cpu_usage", Value: 95.2, Threshold: 90,
		Summary: "CPU 使用率 95.2% 超过阈值 90%", FiredAt: "2026-10-20 10:00:00",
	}
}

// fakeServer 用自签 TLS 模拟官方机器人域名，注入跳过证书校验的 client
func fakeServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	originalHost, originalClient := checkHost, client
	checkHost = func(Kind, string) bool { return true }
	client = &http.Client{
		Timeout: 5 * time.Second,
		// 仅测试用：httptest 自签证书，生产 client 始终校验
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	t.Cleanup(func() {
		srv.Close()
		checkHost, client = originalHost, originalClient
	})
	return srv
}

func TestDispatchDingTalkSignAndBody(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	var gotBody map[string]interface{}
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	})
	_, err := Dispatch(context.Background(), DingTalk, samplePayload(), srv.URL+"/robot/send?access_token=abc", "SEC123", 1, nil)
	if err != nil {
		t.Fatalf("钉钉投递失败: %v", err)
	}
	if gotPath != "/robot/send" || gotQuery.Get("access_token") != "abc" {
		t.Fatalf("URL 被破坏: path=%s query=%v", gotPath, gotQuery)
	}
	if gotQuery.Get("timestamp") == "" || gotQuery.Get("sign") == "" {
		t.Fatalf("缺少加签参数: %v", gotQuery)
	}
	if decoded, signErr := url.QueryUnescape(gotQuery.Get("sign")); signErr != nil || decoded == "" {
		t.Fatalf("sign 无法解析: %v", gotQuery.Get("sign"))
	}
	if gotBody["msgtype"] != "markdown" {
		t.Fatalf("msgtype=%v", gotBody["msgtype"])
	}
	md, _ := gotBody["markdown"].(map[string]interface{})
	if md == nil || !strings.Contains(md["title"].(string), "CPU 使用率过高") || !strings.Contains(md["text"].(string), "10.0.0.8") {
		t.Fatalf("消息体不完整: %v", md)
	}
}

const base64Prefix = ""

func TestDispatchWeComNoSecret(t *testing.T) {
	var gotBody map[string]interface{}
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	})
	if _, err := Dispatch(context.Background(), WeCom, samplePayload(), srv.URL+"/cgi-bin/webhook/send?key=k1", "", 1, nil); err != nil {
		t.Fatalf("企微投递失败: %v", err)
	}
	if gotBody["msgtype"] != "markdown" {
		t.Fatalf("msgtype=%v", gotBody["msgtype"])
	}
}

func TestDispatchFeishuSignHeadersAndCard(t *testing.T) {
	var gotHeader http.Header
	var gotBody map[string]interface{}
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"code":0,"msg":"success"}`))
	})
	if _, err := Dispatch(context.Background(), Feishu, samplePayload(), srv.URL+"/open-apis/bot/v2/hook/xyz", "fssecret", 1, nil); err != nil {
		t.Fatalf("飞书投递失败: %v", err)
	}
	if gotHeader.Get("X-Lark-Signature") == "" || gotHeader.Get("X-Lark-Request-Timestamp") == "" || gotHeader.Get("X-Lark-Request-Nonce") == "" {
		t.Fatalf("缺少飞书签名头: %v", gotHeader)
	}
	if gotBody["msg_type"] != "interactive" {
		t.Fatalf("msg_type=%v", gotBody["msg_type"])
	}
}

func TestDispatchBusinessErrorRetries(t *testing.T) {
	calls := 0
	var attempts []webhook.Attempt
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"errcode":310000,"errmsg":"keywords not in content"}`))
	})
	last, err := Dispatch(context.Background(), DingTalk, samplePayload(), srv.URL+"/robot/send", "", webhook.MaxAttempts, func(a webhook.Attempt) {
		attempts = append(attempts, a)
	})
	if err == nil {
		t.Fatal("业务码失败应返回错误")
	}
	if calls != webhook.MaxAttempts {
		t.Fatalf("应重试 %d 次, 实际 %d", webhook.MaxAttempts, calls)
	}
	if last.Success || !strings.Contains(last.ErrorMsg, "310000") {
		t.Fatalf("最后一次尝试结果不对: %+v", last)
	}
	if len(attempts) != webhook.MaxAttempts {
		t.Fatalf("每次尝试都应回调, got %d", len(attempts))
	}
}

func TestDispatchErrcodeZeroSuccess(t *testing.T) {
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":0}`))
	})
	attempt, err := Dispatch(context.Background(), DingTalk, samplePayload(), srv.URL+"/x", "", 1, nil)
	if err != nil || !attempt.Success {
		t.Fatalf("errcode=0 应成功: %+v %v", attempt, err)
	}
}

func TestValidateUrlOfficialHost(t *testing.T) {
	if err := ValidateUrl(DingTalk, "https://oapi.dingtalk.com/robot/send?access_token=x"); err != nil {
		t.Fatalf("官方域名应通过: %v", err)
	}
	if err := ValidateUrl(DingTalk, "https://10.0.0.1/robot/send"); err == nil {
		t.Fatal("内网地址应拒绝（防 SSRF）")
	}
	if err := ValidateUrl(WeCom, "http://qyapi.weixin.qq.com/x"); err == nil {
		t.Fatal("明文 http 应拒绝")
	}
	if err := ValidateUrl(Feishu, "https://open.feishu.cn/open-apis/bot/v2/hook/x"); err != nil {
		t.Fatalf("飞书官方域名应通过: %v", err)
	}
}
