package logs

import (
	"strings"
	"testing"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/logs/datasource"
)

func TestNormalizeAlertRule(t *testing.T) {
	rule := &models.OpsLogAlertRule{Name: "  错误日志突增  ", Level: "error", Keyword: "oom", Host: `10.0.0."1`}
	if err := normalizeAlertRule(rule); err != nil {
		t.Fatalf("正常规则校验失败: %v", err)
	}
	if rule.Name != "错误日志突增" {
		t.Errorf("名称未去除首尾空格: %q", rule.Name)
	}
	if rule.Level != "ERROR" {
		t.Errorf("级别未统一大写: %q", rule.Level)
	}
	if rule.Host != "" {
		t.Errorf("含引号的标签值应被清空: %q", rule.Host)
	}
	if rule.WindowSec != 300 || rule.Threshold != 10 || rule.CooldownSec != 300 {
		t.Errorf("缺省值填充错误: window=%d threshold=%d cooldown=%d", rule.WindowSec, rule.Threshold, rule.CooldownSec)
	}
	if rule.Severity != models.SeverityWarning {
		t.Errorf("默认告警级别错误: %d", rule.Severity)
	}
	if rule.GroupByHost != "1" {
		t.Errorf("默认应按主机分组: %q", rule.GroupByHost)
	}

	cases := map[string]*models.OpsLogAlertRule{
		"空名称":   {Name: "  "},
		"窗口过小":  {Name: "r", WindowSec: 10},
		"窗口过大":  {Name: "r", WindowSec: 7200},
		"未知级别":  {Name: "r", Level: "SEVERE"},
		"关键字换行": {Name: "r", Keyword: "a\nb"},
		"选择器非法": {Name: "r", Selector: "job=ops_log"},
	}
	for name, bad := range cases {
		if err := normalizeAlertRule(bad); err == nil {
			t.Errorf("%s 应校验失败", name)
		}
	}
}

func TestBuildAlertLogQL(t *testing.T) {
	rule := &models.OpsLogAlertRule{
		Name:        "错误日志突增",
		Host:        "10.0.0.1",
		Level:       "ERROR",
		Keyword:     `disk "full"`,
		WindowSec:   300,
		Threshold:   10,
		GroupByHost: "1",
	}
	got, err := BuildAlertLogQL(rule)
	if err != nil {
		t.Fatalf("生成表达式失败: %v", err)
	}
	want := `sum by (host) (count_over_time({host="10.0.0.1",job="ops_log",level="ERROR"} |= "disk \"full\"" [300s]))`
	if got != want {
		t.Fatalf("按主机分组表达式错误\n got=%s\nwant=%s", got, want)
	}

	rule.GroupByHost = "0"
	if got, err = BuildAlertLogQL(rule); err != nil {
		t.Fatalf("生成表达式失败: %v", err)
	}
	if !strings.HasPrefix(got, "sum(count_over_time(") || !strings.HasSuffix(got, "[300s]))") {
		t.Fatalf("汇总表达式错误: %s", got)
	}

	// 未填任何条件时兜底到 job="ops_log"，不会退化成全量扫描
	if got, err = BuildAlertLogQL(&models.OpsLogAlertRule{WindowSec: 60}); err != nil {
		t.Fatalf("空条件应兜底: %v", err)
	}
	if !strings.Contains(got, `{job="ops_log"}`) {
		t.Fatalf("默认 job 标签缺失: %s", got)
	}

	custom := &models.OpsLogAlertRule{Selector: `{job="ops_log",source="nginx"}`, WindowSec: 60}
	if got, err = BuildAlertLogQL(custom); err != nil {
		t.Fatalf("高级选择器应允许: %v", err)
	}
	if !strings.HasPrefix(got, "sum(count_over_time({job=\"ops_log\",source=\"nginx\"} [60s]))") {
		t.Fatalf("未使用自定义选择器: %s", got)
	}
}

func TestAlertMuteReason(t *testing.T) {
	now := time.Now()
	rule := &models.OpsLogAlertRule{ChannelIds: "1"}
	if reason := muteReason(rule, now); reason != "" {
		t.Errorf("正常规则不应抑制: %s", reason)
	}

	rule.Muted = "1"
	if reason := muteReason(rule, now); reason == "" {
		t.Error("已静音规则应给出抑制原因")
	}
	rule.Muted = "0"

	rule.SilenceUntil = now.Add(time.Minute).Unix()
	if reason := muteReason(rule, now); reason == "" {
		t.Error("静默期内的规则应给出抑制原因")
	}
	rule.SilenceUntil = 0

	rule.ChannelIds = ""
	if reason := muteReason(rule, now); reason == "" {
		t.Error("未配置通知渠道的规则应给出抑制原因")
	}
}

func TestAlertFingerprintAndSummary(t *testing.T) {
	labels := map[string]string{"host": "10.0.0.1", "job": "ops_log"}
	if fingerprint(7, labels) != fingerprint(7, map[string]string{"job": "ops_log", "host": "10.0.0.1"}) {
		t.Error("指纹应与标签顺序无关")
	}
	if fingerprint(7, labels) == fingerprint(8, labels) {
		t.Error("不同规则的指纹不应相同")
	}

	summary := buildSummary(&models.OpsLogAlertRule{Name: "错误日志突增", WindowSec: 300, Threshold: 10},
		datasource.VectorSample{Labels: labels, Value: 25})
	for _, want := range []string{"300 秒内命中 25 条", "阈值 10", "10.0.0.1"} {
		if !strings.Contains(summary, want) {
			t.Errorf("摘要缺少 %q: %s", want, summary)
		}
	}

	long := &models.OpsLogAlertRule{Base: models.Base{ID: 7}, Name: strings.Repeat("长", 600), WindowSec: 60, Threshold: 1}
	if got := buildSummary(long, datasource.VectorSample{Value: 2}); len([]rune(got)) > 512 {
		t.Errorf("摘要未按 512 字符截断: %d", len([]rune(got)))
	}
}
