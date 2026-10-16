package datasource

import (
	"crypto/tls"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// LokiURL Loki 地址（日志写入与检索端）
func LokiURL() string {
	return strings.TrimRight(envStr("LOKI_URL", "http://localhost:3100"), "/")
}

// lokiToken Loki 的 Bearer Token，单进程免租户部署时可为空
func lokiToken() string {
	return strings.TrimSpace(os.Getenv("LOKI_TOKEN"))
}

// lokiTimeout 访问 Loki 的超时时间(秒)
func lokiTimeout() time.Duration {
	return time.Duration(envInt("LOKI_TIMEOUT", 15)) * time.Second
}

// lokiInsecure 自建证书场景下跳过证书校验
func lokiInsecure() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("LOKI_INSECURE_SKIP_VERIFY")))
	return val == "1" || val == "true"
}

// LogMaxBytes 单轮从远端主机拉取的日志字节上限
func LogMaxBytes() int {
	return envInt("LOG_COLLECT_MAX_BYTES", 512*1024)
}

// CollectIntervalSec 日志采集轮询间隔(秒)
func CollectIntervalSec() int {
	return envInt("LOG_COLLECT_INTERVAL", 15)
}

// CollectConcurrency 单轮并发拉取的主机数
func CollectConcurrency() int {
	return envInt("LOG_COLLECT_CONCURRENCY", 5)
}

// CollectTimeoutSec 单台主机拉取超时(秒)
func CollectTimeoutSec() int {
	return envInt("LOG_COLLECT_TIMEOUT", 20)
}

// InitialBackBytes 首次采集时从文件末尾回溯的字节数，0 表示从头开始
func InitialBackBytes() int {
	return envInt("LOG_COLLECT_INITIAL_BACK", 64*1024)
}

func newLokiHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: lokiInsecure()},
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

// EnvInt 读取正整数环境变量，供上层服务复用同一套取值规则
func EnvInt(key string, def int) int { return envInt(key, def) }
