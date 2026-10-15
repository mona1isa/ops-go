package datasource

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakePrometheus 返回真实 Prometheus HTTP API 的响应结构
func fakePrometheus(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/-/ready":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/query":
			got := r.URL.Query().Get("query")
			if got == "" {
				t.Errorf("瞬时查询缺少 query 参数")
			}
			// 表达式非法时 Prometheus 返回 400 + error 字段
			if got == "bad(" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"error","errorType":"bad_data","error":"invalid parameter"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[
				{"metric":{"__name__":"ops_cpu_usage_percent","instance":"10.0.0.1","job":"ops_node"},"value":[1700000000.5,"87.30"]},
				{"metric":{"instance":"10.0.0.2","mountpoint":"/data","fstype":"xfs"},"value":[1700000000.5,"91.50"]},
				{"metric":{"instance":"10.0.0.3"},"value":[1700000000.5,"NaN"]}
			]}}`))
		case "/api/v1/query_range":
			q := r.URL.Query()
			for _, key := range []string{"query", "start", "end", "step"} {
				if q.Get(key) == "" {
					t.Errorf("区间查询缺少 %s 参数", key)
				}
			}
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[
				{"metric":{"instance":"10.0.0.1"},"values":[[1700000000,"10.5"],[1700000060,"20.25"],[1700000120,"33"]] }
			]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newTestClient(base string) *PrometheusClient {
	return &PrometheusClient{base: base, client: &http.Client{Timeout: 2 * time.Second}}
}

func TestPrometheusQuery(t *testing.T) {
	srv := fakePrometheus(t)
	defer srv.Close()

	ctx := context.Background()
	c := newTestClient(srv.URL)

	if !c.Available(ctx) {
		t.Fatal("数据源应判定为可用")
	}

	samples, err := c.Query(ctx, `ops_cpu_usage_percent > 85`)
	if err != nil {
		t.Fatalf("瞬时查询失败: %v", err)
	}
	if len(samples) != 3 {
		t.Fatalf("期望 3 条样本，实际 %d", len(samples))
	}
	if samples[0].Instance() != "10.0.0.1" || samples[0].Value != 87.3 {
		t.Fatalf("样本解析错误: %+v", samples[0])
	}
	if samples[0].Timestamp != 1700000000 {
		t.Fatalf("时间戳应转为秒: %+v", samples[0])
	}
	if samples[1].Labels["mountpoint"] != "/data" {
		t.Fatalf("标签解析错误: %+v", samples[1])
	}
	// Prometheus 对 NaN 返回字符串 "NaN"，转换后仍是 NaN，由调用方过滤
	if !isNaN(samples[2].Value) {
		t.Fatalf("NaN 样本应保留原值供上层判断: %+v", samples[2])
	}
}

func TestPrometheusQueryRange(t *testing.T) {
	srv := fakePrometheus(t)
	defer srv.Close()

	c := newTestClient(srv.URL)
	series, err := c.QueryRange(context.Background(), "ops_cpu_usage_percent",
		time.Unix(1700000000, 0), time.Unix(1700000120, 0), time.Minute)
	if err != nil {
		t.Fatalf("区间查询失败: %v", err)
	}
	if len(series) != 1 || len(series[0].Points) != 3 {
		t.Fatalf("序列解析错误: %+v", series)
	}
	if series[0].Points[1].Value != 20.25 || series[0].Points[1].Timestamp != 1700000060 {
		t.Fatalf("数据点解析错误: %+v", series[0].Points[1])
	}
	if series[0].Labels["instance"] != "10.0.0.1" {
		t.Fatalf("序列标签解析错误: %+v", series[0].Labels)
	}
}

func TestPrometheusQueryErrors(t *testing.T) {
	srv := fakePrometheus(t)
	defer srv.Close()

	ctx := context.Background()

	// 表达式非法：Prometheus 返回 400 与 error 字段
	if _, err := newTestClient(srv.URL).Query(ctx, "bad("); err == nil {
		t.Fatal("400 响应应返回错误")
	} else if !strings.Contains(err.Error(), "invalid parameter") {
		t.Fatalf("应透出 Prometheus 的错误描述: %v", err)
	}

	// 服务不可达：包装成数据源不可用，供接口层给出统一提示
	dead := newTestClient("http://127.0.0.1:1")
	if _, err := dead.Query(ctx, "up"); !errors.Is(err, ErrDatasourceUnavailable) {
		t.Fatalf("期望 ErrDatasourceUnavailable，实际 %v", err)
	}
	if dead.Available(ctx) {
		t.Fatal("不可达数据源应判定为不可用")
	}
}

func isNaN(f float64) bool {
	return f != f
}
