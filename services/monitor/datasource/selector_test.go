package datasource

import (
	"strings"
	"testing"
)

func TestSelectorBuildsValidPromQL(t *testing.T) {
	expr, err := Selector(MetricDiskUsage, map[string]string{
		"instance":   "10.0.0.8",
		"mountpoint": "/data",
		"device":     "",
	})
	if err != nil {
		t.Fatalf("构造选择器失败: %v", err)
	}
	if expr != `ops_disk_usage_percent{instance="10.0.0.8",mountpoint="/data"}` {
		t.Errorf("表达式 = %s", expr)
	}
}

func TestSelectorWithoutLabels(t *testing.T) {
	expr, err := Selector(MetricCPUUsage, nil)
	if err != nil {
		t.Fatalf("构造选择器失败: %v", err)
	}
	if expr != MetricCPUUsage {
		t.Errorf("表达式 = %s", expr)
	}
}

// 标签值来自请求参数，必须阻止逃逸出选择器语法
func TestSelectorRejectsInjection(t *testing.T) {
	cases := []map[string]string{
		{"instance": `1.1.1.1"} or {__name__=~".+"`},
		{"instance": "1.1.1.1\""},
		{"mountpoint": "/data; drop"},
		{"device": "eth0\n0"},
		{"device": "eth0{}"},
		{"instance": strings.Repeat("a", 200)},
	}
	for _, labels := range cases {
		if _, err := Selector(MetricCPUUsage, labels); err == nil {
			t.Errorf("标签 %+v 应当被拒绝", labels)
		}
	}

	badMetrics := []string{"", "1abc", "ops cpu", "ops;drop", strings.Repeat("m", 70)}
	for _, metric := range badMetrics {
		if _, err := Selector(metric, map[string]string{"instance": "1.1.1.1"}); err == nil {
			t.Errorf("指标名 %q 应当被拒绝", metric)
		}
	}
	if _, err := Selector(MetricCPUUsage, map[string]string{"1bad": "x"}); err == nil {
		t.Error("非法标签名应当被拒绝")
	}
}

func TestValidPanelMetric(t *testing.T) {
	for _, name := range PanelMetricNames() {
		if !ValidPanelMetric(name) {
			t.Errorf("%s 应当是面板可用指标", name)
		}
		if PanelMetricLabel(name) == "" || PanelMetricLabel(name) == name {
			t.Errorf("%s 缺少中文展示名", name)
		}
	}
	if ValidPanelMetric("node_cpu_seconds_total") {
		t.Error("未采集的指标不应出现在白名单")
	}
}

func TestInstantExpr(t *testing.T) {
	expr, err := InstantExpr("10.0.0.8")
	if err != nil {
		t.Fatalf("构造快照表达式失败: %v", err)
	}
	if expr != `{__name__=~"ops_.+",instance="10.0.0.8"}` {
		t.Errorf("表达式 = %s", expr)
	}
	for _, bad := range []string{"", `10.0.0.8"`, "10.0.0.8 or", "a{b}"} {
		if _, err := InstantExpr(bad); err == nil {
			t.Errorf("主机地址 %q 应当被拒绝", bad)
		}
	}
}
