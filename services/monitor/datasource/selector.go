package datasource

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// 面板可选指标：由采集脚本直接写入的百分比/瞬时值 gauge
var panelMetrics = map[string]string{
	MetricCPUUsage:  "CPU 使用率(%)",
	MetricMemUsage:  "内存使用率(%)",
	MetricLoad1:     "1 分钟负载",
	MetricUptime:    "运行时长(秒)",
	MetricDiskUsage: "磁盘使用率(%)",
	MetricNetRx:     "网络接收(B/s)",
	MetricNetTx:     "网络发送(B/s)",
	MetricHostInfo:  "主机信息",
}

var (
	metricPattern = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]{0,63}$`)
	// labelValuePattern 允许 IPv4/IPv6、挂载点、网卡名等安全字符，排除引号与花括号等选择器语法
	labelValuePattern = regexp.MustCompile(`^[A-Za-z0-9_\-.:/]{1,128}$`)
	labelNamePattern  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,63}$`)
)

// PanelMetricNames 返回面板允许查询的指标名，按字典序排列
func PanelMetricNames() []string {
	names := make([]string, 0, len(panelMetrics))
	for name := range panelMetrics {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// PanelMetricLabel 指标中文名，未知指标返回指标名本身
func PanelMetricLabel(name string) string {
	if label, ok := panelMetrics[name]; ok {
		return label
	}
	return name
}

// ValidPanelMetric 判断指标是否在面板白名单内
func ValidPanelMetric(name string) bool {
	_, ok := panelMetrics[name]
	return ok
}

// Selector 构造带标签选择器的 PromQL 片段，所有值经过字符白名单校验，杜绝表达式注入
func Selector(metric string, labels map[string]string) (string, error) {
	if !metricPattern.MatchString(metric) {
		return "", errors.New("指标名称不合法")
	}
	if len(labels) == 0 {
		return metric, nil
	}

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if !labelNamePattern.MatchString(key) {
			return "", fmt.Errorf("标签名称不合法: %s", key)
		}
		value := strings.TrimSpace(labels[key])
		if value == "" {
			continue
		}
		if !labelValuePattern.MatchString(value) {
			return "", fmt.Errorf("标签值不合法: %s=%s", key, value)
		}
		parts = append(parts, fmt.Sprintf(`%s="%s"`, key, value))
	}
	if len(parts) == 0 {
		return metric, nil
	}
	return metric + "{" + strings.Join(parts, ",") + "}", nil
}

// InstantExpr 构造全量指标查询表达式，用于主机快照
func InstantExpr(instance string) (string, error) {
	if !labelValuePattern.MatchString(strings.TrimSpace(instance)) {
		return "", errors.New("主机地址不合法")
	}
	return fmt.Sprintf(`{__name__=~"%s",instance="%s"}`, "ops_.+", strings.TrimSpace(instance)), nil
}
