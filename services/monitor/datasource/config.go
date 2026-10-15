package datasource

import (
	"crypto/tls"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// MetricJob 指标分组名，Pushgateway 的 URL 路径与 Prometheus 抓取任务需保持一致
const MetricJob = "ops_node"

// PrometheusURL Prometheus 地址（面板查询与告警评估的数据源）
func PrometheusURL() string {
	return strings.TrimRight(envStr("PROMETHEUS_URL", "http://localhost:9090"), "/")
}

// PushgatewayURL Pushgateway 地址（采集结果的写入端）
func PushgatewayURL() string {
	return strings.TrimRight(envStr("PUSHGATEWAY_URL", "http://localhost:9091"), "/")
}

// prometheusToken Prometheus 只读接口的 Bearer Token，可为空
func prometheusToken() string {
	return strings.TrimSpace(os.Getenv("PROMETHEUS_TOKEN"))
}

// pushgatewayToken Pushgateway 的 Bearer Token，可为空
func pushgatewayToken() string {
	return strings.TrimSpace(os.Getenv("PUSHGATEWAY_TOKEN"))
}

// pushTimeout 推送/删除指标分组的超时时间
func pushTimeout() time.Duration {
	return time.Duration(envInt("MONITOR_PUSH_TIMEOUT", 10)) * time.Second
}

// insecureSkipVerify 自建证书场景下跳过证书校验的开关
func insecureSkipVerify() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("PROMETHEUS_INSECURE_SKIP_VERIFY")))
	return val == "1" || val == "true"
}

// newHTTPClient 构造访问监控数据源的 HTTP 客户端
func newHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify()},
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func envStr(key, def string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return def
}

func envInt(key string, def int) int {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if num, err := strconv.Atoi(val); err == nil && num > 0 {
			return num
		}
	}
	return def
}
