package monitor

import (
	"os"
	"strconv"
	"strings"
)

// envInt 读取整型环境变量，非法值返回默认值
func envInt(key string, def int) int {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if num, err := strconv.Atoi(val); err == nil && num > 0 {
			return num
		}
	}
	return def
}

// collectIntervalSec 采集轮询间隔（秒）
func collectIntervalSec() int { return envInt("MONITOR_COLLECT_INTERVAL", 60) }

// collectTimeoutSec 单台主机采集超时（秒）
func collectTimeoutSec() int { return envInt("MONITOR_COLLECT_TIMEOUT", 15) }

// collectConcurrency 采集并发上限
func collectConcurrency() int { return envInt("MONITOR_CONCURRENCY", 20) }

// evalIntervalSec 告警规则评估间隔（秒）
func evalIntervalSec() int { return envInt("MONITOR_EVAL_INTERVAL", 30) }

// pushTimeoutSec 指标写入/清理数据源的超时（秒）
func pushTimeoutSec() int { return envInt("MONITOR_PUSH_TIMEOUT", 10) }

// queryTimeoutSec 单次监控数据查询超时（秒）
func queryTimeoutSec() int { return envInt("MONITOR_QUERY_TIMEOUT", 15) }

// webhookTimeoutSec 通知回调超时（秒）
func webhookTimeoutSec() int { return envInt("WEBHOOK_TIMEOUT", 10) }

// truncate 按字符数截断并去除换行，用于错误信息与响应体入库
func truncate(s string, max int) string {
	s = strings.NewReplacer("\n", " ", "\r", " ").Replace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
