package logs

import (
	"testing"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/logs/datasource"
)

func TestBuildStreams(t *testing.T) {
	source := &models.OpsLogSource{
		Job:        "ops_log",
		InstanceIp: "10.0.0.1",
		Source:     "nginx",
		Path:       "/var/log/app/*.log",
		Labels:     `{"env":"prod","bad\"label":"x"}`,
		LevelRegex: "",
	}
	lines := []string{
		"2026-10-16 10:00:01 ERROR boom",
		"2026-10-16 10:00:02 INFO ok",
		"plain line without time",
	}

	streams := buildStreams(source, lines)
	if len(streams) != 3 {
		t.Fatalf("应按级别拆流: got=%d", len(streams))
	}

	byLevel := map[string]datasource.Stream{}
	for _, stream := range streams {
		byLevel[stream.Labels["level"]] = stream
	}
	if len(byLevel) != 3 {
		t.Fatalf("级别分组错误: %v", byLevel)
	}
	for level, stream := range byLevel {
		if len(stream.Entries) != 1 {
			t.Errorf("%s 流行数错误: %d", level, len(stream.Entries))
		}
		if stream.Labels["job"] != "ops_log" || stream.Labels["host"] != "10.0.0.1" {
			t.Errorf("%s 基础标签错误: %v", level, stream.Labels)
		}
		if stream.Labels["source"] != "nginx" || stream.Labels["path"] != "/var/log/app/*.log" {
			t.Errorf("%s 来源标签错误: %v", level, stream.Labels)
		}
		if stream.Labels["env"] != "prod" {
			t.Errorf("%s 自定义标签未注入: %v", level, stream.Labels)
		}
		if _, bad := stream.Labels[`bad"label`]; bad {
			t.Errorf("%s 非法标签名未被丢弃: %v", level, stream.Labels)
		}
	}

	// 行内时间优先，无时间的行落到采集时间
	if got := byLevel["ERROR"].Entries[0].Timestamp.Format("2006-01-02 15:04:05"); got != "2026-10-16 10:00:01" {
		t.Errorf("行内时间解析错误: %s", got)
	}
	if ts := byLevel[""].Entries[0].Timestamp; time.Since(ts) > time.Minute {
		t.Errorf("无时间行应回落到采集时间: %s", ts)
	}

	// 空标签值不应写入流，避免 Loki 拒绝
	tail := buildStreams(&models.OpsLogSource{Job: "ops_log", InstanceIp: "10.0.0.1"}, []string{"no level here"})
	if len(tail) != 1 || len(tail[0].Labels) != 2 {
		t.Fatalf("空来源/路径标签应被忽略: %+v", tail)
	}
}

func TestBuildStreamsReservedLabel(t *testing.T) {
	source := &models.OpsLogSource{
		Job:        "ops_log",
		InstanceIp: "10.0.0.1",
		Path:       "/var/log/a.log",
		Labels:     `{"host":"evil","env":"prod","a-b":"x","9bad":"y","empty":""}`,
	}
	streams := buildStreams(source, []string{"2026-10-16 10:00:01 ERROR boom"})
	if len(streams) != 1 {
		t.Fatalf("流数量错误: %d", len(streams))
	}
	labels := streams[0].Labels
	if labels["host"] != "10.0.0.1" {
		t.Errorf("保留标签 host 被自定义标签覆盖: %v", labels)
	}
	if labels["env"] != "prod" {
		t.Errorf("合法自定义标签未注入: %v", labels)
	}
	for _, key := range []string{"a-b", "9bad", "empty"} {
		if _, exists := labels[key]; exists {
			t.Errorf("非法自定义标签应被丢弃: %s=%v", key, labels[key])
		}
	}

	if got := parseSourceLabels("not json"); len(got) != 0 {
		t.Errorf("非法 JSON 应返回空标签集: %v", got)
	}
	if got := parseSourceLabels(""); len(got) != 0 {
		t.Errorf("空配置应返回空标签集: %v", got)
	}
}

func TestNormalizeLabels(t *testing.T) {
	raw := `{"env":"prod","host":"evil","9bad":"x","a-b":"x","empty":"","ok_key":"v1"}`
	got := normalizeLabels(raw)
	want := `{"env":"prod","ok_key":"v1"}`
	if got != want {
		t.Fatalf("标签规范化结果错误\n got=%s\nwant=%s", got, want)
	}
	for _, empty := range []string{"", "  ", "not json", `{}`, `{"host":"x"}`} {
		if normalizeLabels(empty) != "" {
			t.Errorf("应规范化为空串: %q -> %q", empty, normalizeLabels(empty))
		}
	}
}

func TestPickDueSources(t *testing.T) {
	now := time.Now()
	older := now.Add(-2 * time.Minute)
	recent := now.Add(-5 * time.Second)

	a := models.OpsLogSource{Base: models.Base{ID: 1}, IntervalSec: 60}
	b := models.OpsLogSource{Base: models.Base{ID: 2}, IntervalSec: 60, LastCollectAt: &older}
	c := models.OpsLogSource{Base: models.Base{ID: 3}, IntervalSec: 60, LastCollectAt: &recent}
	d := models.OpsLogSource{Base: models.Base{ID: 4}, IntervalSec: 0}

	due := pickDueSources([]models.OpsLogSource{a, b, c, d}, now, nil, 30)
	if len(due) != 3 {
		t.Fatalf("到期采集源数量错误: %v", due)
	}
	for _, item := range due {
		if item.ID == 3 {
			t.Fatal("未到期的采集源不应被拉取")
		}
	}

	// 本轮尝试时间比库里的成功时间更新时，同样按间隔退避
	attempts := map[int]time.Time{1: now.Add(-10 * time.Second)}
	if got := pickDueSources([]models.OpsLogSource{a}, now, attempts, 60); len(got) != 0 {
		t.Errorf("失败退避未生效: %v", got)
	}
	// IntervalSec 为 0 时回落到全局间隔
	if got := pickDueSources([]models.OpsLogSource{d}, now, nil, 30); len(got) != 1 {
		t.Errorf("缺省间隔未生效: %v", got)
	}
}

func TestShellQuote(t *testing.T) {
	if got := shellQuote("/var/log/a.log"); got != "'/var/log/a.log'" {
		t.Errorf("普通路径引号错误: %s", got)
	}
	if got := shellQuote(`a'; rm -rf / #`); got != `'a'\''; rm -rf / #'` {
		t.Errorf("单引号转义错误: %s", got)
	}
}
