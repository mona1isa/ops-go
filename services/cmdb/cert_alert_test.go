package cmdb

import (
	"strings"
	"testing"
	"time"

	"github.com/zhany/ops-go/models"
)

// 本测试仅覆盖证书到期告警的纯函数逻辑，不依赖数据库

func TestNormalizeCertRule(t *testing.T) {
	// 名称为空应报错
	rule := &models.OpsCertAlertRule{Name: "  "}
	if err := normalizeCertRule(rule); err == nil {
		t.Fatal("空规则名称应报错")
	}

	// 默认值补齐：warning_days、cooldown、severity 落到合法范围
	rule = &models.OpsCertAlertRule{Name: "默认规则"}
	if err := normalizeCertRule(rule); err != nil {
		t.Fatalf("合法规则不应报错: %v", err)
	}
	if rule.WarningDays != 30 {
		t.Errorf("WarningDays 期望 30，实际 %d", rule.WarningDays)
	}
	if rule.CooldownSec != 86400 {
		t.Errorf("CooldownSec 期望 86400，实际 %d", rule.CooldownSec)
	}
	if rule.Severity != models.SeverityWarning {
		t.Errorf("Severity 期望默认警告(2)，实际 %d", rule.Severity)
	}
	if rule.Enabled != "1" {
		t.Errorf("Enabled 期望默认启用(1)，实际 %q", rule.Enabled)
	}
	if rule.Muted != "0" {
		t.Errorf("Muted 期望默认不静音(0)，实际 %q", rule.Muted)
	}

	// 超范围预警天数应报错
	rule = &models.OpsCertAlertRule{Name: "过长", WarningDays: 99999}
	if err := normalizeCertRule(rule); err == nil {
		t.Error("预警天数超过 3650 应报错")
	}

	// 非法模型类型被过滤，只保留已知类型
	rule = &models.OpsCertAlertRule{Name: "类型过滤", ModelTypes: "ssl, bogus ,domain,,"}
	if err := normalizeCertRule(rule); err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if rule.ModelTypes != "ssl,domain" {
		t.Errorf("ModelTypes 期望 %q，实际 %q", "ssl,domain", rule.ModelTypes)
	}
}

func TestSplitModelTypes(t *testing.T) {
	got := splitModelTypes(" ssl , ,domain,")
	want := []string{"ssl", "domain"}
	if len(got) != len(want) {
		t.Fatalf("期望 %d 项，实际 %d 项：%v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 项期望 %q，实际 %q", i, want[i], got[i])
		}
	}
	if len(splitModelTypes("")) != 0 {
		t.Error("空串应返回空切片")
	}
}

func TestCertMuteReason(t *testing.T) {
	now := time.Now()

	if reason := certMuteReason(&models.OpsCertAlertRule{ChannelIds: "1", Muted: "1"}, now); reason != "规则已静音" {
		t.Errorf("静音判定错误: %q", reason)
	}
	if reason := certMuteReason(&models.OpsCertAlertRule{ChannelIds: "1", SilenceUntil: now.Add(time.Hour).Unix()}, now); reason != "处于静默期" {
		t.Errorf("静默期判定错误: %q", reason)
	}
	if reason := certMuteReason(&models.OpsCertAlertRule{ChannelIds: "  "}, now); reason != "未配置通知渠道" {
		t.Errorf("未配置渠道判定错误: %q", reason)
	}
	// 已静默期结束后应可通知
	if reason := certMuteReason(&models.OpsCertAlertRule{ChannelIds: "1", SilenceUntil: now.Add(-time.Hour).Unix()}, now); reason != "" {
		t.Errorf("静默到期后应允许通知，实际: %q", reason)
	}
	// 正常可通知
	if reason := certMuteReason(&models.OpsCertAlertRule{ChannelIds: "1,2"}, now); reason != "" {
		t.Errorf("正常规则不应被抑制，实际: %q", reason)
	}
}

func TestCertFingerprintStable(t *testing.T) {
	a := certFingerprint(1, 100)
	b := certFingerprint(1, 100)
	if a != b {
		t.Error("同规则同资产的指纹必须稳定")
	}
	if a == certFingerprint(1, 101) || a == certFingerprint(2, 100) {
		t.Error("规则或资产不同，指纹应不同")
	}
	if len(a) != 64 {
		t.Errorf("指纹应为 sha256 hex(64)，实际长度 %d", len(a))
	}
}

func TestDaysUntilAndLevel(t *testing.T) {
	now := time.Now()

	// 到期时间在未来 10 天 → 剩余约 10 天，级别关注(3)
	future := now.AddDate(0, 0, 10)
	row := toCertRow(models.OpsCmdbAsset{Name: "c", ModelType: models.CmdbModelSsl, ExpiredAt: &future})
	if row.Remaining != 10 {
		t.Errorf("剩余天数期望 10，实际 %d", row.Remaining)
	}
	if row.Level != models.SeverityInfo || row.LevelText != "关注" {
		t.Errorf("10 天到期应为关注，实际 %d/%s", row.Level, row.LevelText)
	}

	// 未来 3 天 → 临期警告(2)
	soon := now.AddDate(0, 0, 3)
	row = toCertRow(models.OpsCmdbAsset{Name: "c", ModelType: models.CmdbModelSsl, ExpiredAt: &soon})
	if row.Level != models.SeverityWarning || row.LevelText != "临期" {
		t.Errorf("3 天到期应为临期，实际 %d/%s", row.Level, row.LevelText)
	}

	// 已过期 2 天 → 紧急(1)，剩余为负
	expired := now.AddDate(0, 0, -2)
	row = toCertRow(models.OpsCmdbAsset{Name: "c", ModelType: models.CmdbModelSsl, ExpiredAt: &expired})
	if row.Level != models.SeverityCritical || row.LevelText != "已过期" {
		t.Errorf("已过期应为紧急，实际 %d/%s", row.Level, row.LevelText)
	}
	if row.Remaining >= 0 {
		t.Errorf("已过期剩余天数应为负，实际 %d", row.Remaining)
	}
}

func TestBuildCertHitSeverityEscalatesWhenExpired(t *testing.T) {
	rule := &models.OpsCertAlertRule{Name: "r", WarningDays: 30, Severity: models.SeverityInfo}

	// 未过期时沿用规则级别（提醒）
	notExpired := time.Now().AddDate(0, 0, 5)
	hit := buildCertHit(rule, models.OpsCmdbAsset{ExpiredAt: &notExpired}, time.Now())
	if hit.severity != models.SeverityInfo {
		t.Errorf("临期应沿用规则级别(3)，实际 %d", hit.severity)
	}

	// 已过期一律升为紧急
	overdue := time.Now().AddDate(0, 0, -1)
	hit = buildCertHit(rule, models.OpsCmdbAsset{ExpiredAt: &overdue}, time.Now())
	if hit.severity != models.SeverityCritical {
		t.Errorf("已过期应升级为紧急(1)，实际 %d", hit.severity)
	}
}

func TestBuildCertSummary(t *testing.T) {
	rule := &models.OpsCertAlertRule{Name: "生产证书预警", WarningDays: 30}
	expireAt := time.Date(2026, 11, 1, 9, 0, 0, 0, time.Local)
	asset := models.OpsCmdbAsset{Name: "example.com", ModelType: models.CmdbModelSsl, ExpiredAt: &expireAt}

	// 临期摘要含「剩余 N 天」
	hit := certHit{asset: asset, remaining: 8}
	summary := buildCertSummary(rule, hit)
	if !strings.Contains(summary, "剩余 8 天") || !strings.Contains(summary, "example.com") {
		t.Errorf("临期摘要不符预期: %s", summary)
	}

	// 已过期摘要含「已过期 N 天」
	hit = certHit{asset: asset, remaining: -3}
	summary = buildCertSummary(rule, hit)
	if !strings.Contains(summary, "已过期 3 天") {
		t.Errorf("过期摘要不符预期: %s", summary)
	}

	// 摘要长度受 512 约束
	longName := strings.Repeat("长", 600)
	hit = certHit{asset: models.OpsCmdbAsset{Name: longName, ModelType: models.CmdbModelSsl, ExpiredAt: &expireAt}, remaining: 1}
	if len([]rune(buildCertSummary(rule, hit))) > 512 {
		t.Error("摘要应被截断到 512 字符以内")
	}
}
