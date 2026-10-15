package datasource

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPushgatewayPushAndDelete(t *testing.T) {
	type recorded struct {
		method string
		path   string
		body   string
		auth   string
	}
	var got []recorded

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, recorded{method: r.Method, path: r.URL.Path, body: string(body), auth: r.Header.Get("Authorization")})
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pgw := &Pushgateway{base: srv.URL, token: "secret-token", client: srv.Client()}
	ctx := context.Background()

	if err := pgw.Push(ctx, "10.0.0.1", "ops_cpu_usage_percent 12.5\n"); err != nil {
		t.Fatalf("推送失败: %v", err)
	}
	if err := pgw.Delete(ctx, "10.0.0.1"); err != nil {
		t.Fatalf("删除失败: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("期望 2 次请求，实际 %d", len(got))
	}
	if want := "/metrics/job/ops_node/instance/10.0.0.1"; got[0].path != want {
		t.Fatalf("推送分组路径错误: got %s want %s", got[0].path, want)
	}
	if got[0].method != http.MethodPut || got[1].method != http.MethodDelete {
		t.Fatalf("请求方法错误: %s / %s", got[0].method, got[1].method)
	}
	if got[0].body != "ops_cpu_usage_percent 12.5\n" {
		t.Fatalf("请求体不匹配: %q", got[0].body)
	}
	for i, rec := range got {
		if rec.auth != "Bearer secret-token" {
			t.Fatalf("第 %d 次请求未携带 Bearer Token: %q", i+1, rec.auth)
		}
	}
}

func TestPushgatewayDeleteToleratesMissingGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "group not found", http.StatusNotFound)
	}))
	defer srv.Close()

	pgw := &Pushgateway{base: srv.URL, client: srv.Client()}
	if err := pgw.Delete(context.Background(), "10.0.0.9"); err != nil {
		t.Fatalf("分组不存在应视为清理成功: %v", err)
	}
}

func TestPushgatewayUnavailable(t *testing.T) {
	pgw := &Pushgateway{
		base:   "http://127.0.0.1:1",
		client: &http.Client{Timeout: 500 * time.Millisecond},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if pgw.Available(ctx) {
		t.Fatal("不可达地址应返回 false")
	}
	if err := pgw.Push(ctx, "10.0.0.1", "x 1"); err == nil {
		t.Fatal("推送失败应返回错误")
	} else if !errors.Is(err, ErrDatasourceUnavailable) {
		t.Fatalf("推送失败应包装为数据源不可用: %v", err)
	}
}
