package webhook

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func init() {
	// 单测不等待退避
	backoff = func(int) time.Duration { return 0 }
}

func readBody(t *testing.T, r *http.Request) []byte {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("读取请求体失败: %v", err)
	}
	return body
}

func TestDispatchSuccessAndSignature(t *testing.T) {
	var gotBody []byte
	var gotSignature, gotEvent, gotContentType, gotCustom string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody = readBody(t, r)
		gotSignature = r.Header.Get(SignatureHeader)
		gotEvent = r.Header.Get(EventHeader)
		gotContentType = r.Header.Get("Content-Type")
		gotCustom = r.Header.Get("X-Team")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	body := []byte(`{"ruleName":"CPU 高"}`)
	attempt, err := Dispatch(context.Background(), &Request{
		Url:     server.URL,
		Secret:  "s3cret",
		Headers: map[string]string{"X-Team": "sre", "Content-Type": "text/plain", "": "x"},
		Body:    body,
		EventId: 42,
	}, nil)
	if err != nil {
		t.Fatalf("投递失败: %v", err)
	}
	if !attempt.Success || attempt.StatusCode != http.StatusAccepted || attempt.Attempt != 1 {
		t.Fatalf("投递结果异常: %+v", attempt)
	}
	if string(gotBody) != string(body) {
		t.Errorf("请求体不匹配: %s", gotBody)
	}
	if !Verify("s3cret", gotBody, gotSignature) {
		t.Errorf("签名校验失败: %s", gotSignature)
	}
	if Verify("wrong", gotBody, gotSignature) {
		t.Error("错误密钥竟然校验通过")
	}
	if gotEvent != "42" {
		t.Errorf("事件头 = %q, 期望 42", gotEvent)
	}
	if !strings.HasPrefix(gotContentType, "application/json") {
		t.Errorf("Content-Type = %q, 自定义头不应覆盖保留头", gotContentType)
	}
	if gotCustom != "sre" {
		t.Errorf("自定义头 = %q, 期望 sre", gotCustom)
	}
}

func TestDispatchRetriesEveryAttempt(t *testing.T) {
	var attempts []Attempt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		readBody(t, r)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	_, err := Dispatch(context.Background(), &Request{Url: server.URL, Body: []byte("{}")}, func(item Attempt) {
		attempts = append(attempts, item)
	})
	if err == nil {
		t.Fatal("全部失败时应当返回错误")
	}
	if len(attempts) != MaxAttempts {
		t.Fatalf("尝试次数 = %d, 期望 %d", len(attempts), MaxAttempts)
	}
	for i, item := range attempts {
		if item.Attempt != i+1 {
			t.Errorf("第 %d 次尝试编号 = %d", i+1, item.Attempt)
		}
		if item.Success || item.StatusCode != http.StatusInternalServerError || item.Body != "boom" {
			t.Errorf("尝试记录不完整: %+v", item)
		}
	}
}

func TestDispatchStopsOnFirstSuccess(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		readBody(t, r)
		calls++
		if calls < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	attempt, err := Dispatch(context.Background(), &Request{Url: server.URL, Body: []byte("{}")}, nil)
	if err != nil {
		t.Fatalf("第二次成功后不应报错: %v", err)
	}
	if calls != 2 || !attempt.Success || attempt.Attempt != 2 {
		t.Fatalf("calls=%d attempt=%+v", calls, attempt)
	}
}

func TestDispatchConnectionRefused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	dead := server.URL
	server.Close()

	var logged []Attempt
	_, err := Dispatch(context.Background(), &Request{Url: dead, Body: []byte("{}")}, func(item Attempt) {
		logged = append(logged, item)
	})
	if err == nil {
		t.Fatal("目标不可达时应当报错")
	}
	if len(logged) != MaxAttempts {
		t.Fatalf("失败也应记录 %d 条日志, 实际 %d", MaxAttempts, len(logged))
	}
	for _, item := range logged {
		if item.Success || item.ErrorMsg == "" {
			t.Errorf("失败记录缺少原因: %+v", item)
		}
	}
}

func TestDispatchRejectsBadURL(t *testing.T) {
	cases := map[string]string{
		"空地址":  "",
		"非法协议": "ftp://example.com/hook",
		"缺少主机": "http:///hook",
		"畸形地址": "http://%zz",
	}
	for name, raw := range cases {
		if _, err := Dispatch(context.Background(), &Request{Url: raw, Body: []byte("{}")}, nil); err == nil {
			t.Errorf("%s 应当被拒绝", name)
		} else if !strings.Contains(err.Error(), "通知地址") {
			t.Errorf("%s 错误信息不明确: %v", name, err)
		}
	}
}

func TestDispatchContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		readBody(t, r)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Dispatch(ctx, &Request{Url: server.URL, Body: []byte("{}")}, nil); err == nil {
		t.Fatal("已取消的上下文不应投递成功")
	} else if !errors.Is(err, context.Canceled) {
		t.Errorf("错误类型 = %v, 期望 context.Canceled", err)
	}
}

func TestBuildPayload(t *testing.T) {
	payload := &Payload{
		EventId:      7,
		RuleName:     "磁盘水位",
		Severity:     2,
		SeverityText: SeverityText(2),
		Status:       "firing",
		Instance:     "10.0.0.8",
		Metric:       "ops_disk_usage_percent",
		Value:        91.5,
		Threshold:    85,
		Labels:       map[string]string{"mountpoint": "/data"},
	}
	body, err := BuildBody(payload)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	for _, want := range []string{`"ruleName":"磁盘水位"`, `"severityText":"警告"`, `"mountpoint":"/data"`, `"recovered":false`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("请求体缺少 %s: %s", want, body)
		}
	}
	if SeverityText(1) != "紧急" || SeverityText(3) != "提醒" || SeverityText(9) != "警告" {
		t.Error("级别描述映射错误")
	}
}
