package cmdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/monitor"
)

// CertAlertEvaluator 证书/域名到期告警评估服务：扫描资产台账中带到期时间的记录，
// 剩余天数进入预警窗口即产生告警事件（写入共享表 ops_monitor_alert_event，source=cert），
// 复用监控告警的通知链路投递。与日志告警评估器同一套状态机（命中/持续提醒/恢复）。
type CertAlertEvaluator struct {
	notifier monitor.NotifierService
	running  atomic.Bool
}

// NewCertAlertEvaluator 创建证书到期告警评估服务
func NewCertAlertEvaluator() *CertAlertEvaluator {
	return &CertAlertEvaluator{}
}

// Start 启动评估定时任务。默认每小时扫描一次，可用 CERT_ALERT_INTERVAL（秒）覆盖
func (e *CertAlertEvaluator) Start() {
	interval := time.Duration(certAlertInterval()) * time.Second
	log.Printf("证书到期告警评估服务已启动，轮询间隔 %v", interval)

	time.Sleep(12 * time.Second)
	e.evaluateOnce()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		e.evaluateOnce()
	}
}

func certAlertInterval() int {
	if val := strings.TrimSpace(os.Getenv("CERT_ALERT_INTERVAL")); val != "" {
		if num, err := strconv.Atoi(val); err == nil && num > 0 {
			return num
		}
	}
	return 3600
}

// certHit 单个资产在一轮评估中的命中快照
type certHit struct {
	asset     models.OpsCmdbAsset
	remaining int // 剩余天数，负数表示已过期
	severity  int8
}

// evaluateOnce 单轮评估：逐条规则扫描到期资产并推进事件状态
func (e *CertAlertEvaluator) evaluateOnce() {
	if e.running.Swap(true) {
		return
	}
	defer e.running.Store(false)

	var rules []models.OpsCertAlertRule
	if err := models.DB.Where("del_flag = ? AND enabled = ?", "0", "1").Find(&rules).Error; err != nil {
		log.Println("证书到期告警：查询规则失败:", err)
		return
	}
	if len(rules) == 0 {
		return
	}

	now := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(rules)*15+30)*time.Second)
	defer cancel()

	for i := range rules {
		e.evaluateRule(ctx, &rules[i], now)
	}
}

// evaluateRule 评估单条规则：命中资产新建/推进事件，不再命中的关闭事件
func (e *CertAlertEvaluator) evaluateRule(ctx context.Context, rule *models.OpsCertAlertRule, now time.Time) {
	assets, err := expiringAssets(rule.ModelTypes, rule.WarningDays, now)
	if err != nil {
		log.Printf("证书到期告警：规则 %d 查询到期资产失败: %v", rule.ID, err)
		return
	}

	hits := make(map[string]certHit, len(assets))
	for i := range assets {
		asset := assets[i]
		if asset.ExpiredAt == nil {
			continue
		}
		hit := buildCertHit(rule, asset, now)
		hits[certFingerprint(rule.ID, asset.ID)] = hit
	}

	var open []models.OpsMonitorAlertEvent
	if err := models.DB.Where("rule_id = ? AND source = ? AND status IN ?",
		rule.ID, models.AlertSourceCert, []int8{models.AlertStatusPending, models.AlertStatusFiring}).
		Find(&open).Error; err != nil {
		log.Printf("证书到期告警：规则 %d 读取未恢复事件失败: %v", rule.ID, err)
		return
	}
	openByFingerprint := make(map[string]*models.OpsMonitorAlertEvent, len(open))
	for i := range open {
		openByFingerprint[open[i].Fingerprint] = &open[i]
	}

	for key, hit := range hits {
		if event, exists := openByFingerprint[key]; exists {
			e.renotify(ctx, rule, event, hit, now)
			continue
		}
		e.fire(ctx, rule, key, hit, now)
	}

	// 资产已续期到窗口之外、被删除或退役：关闭历史事件，续期后再次临近会重新告警
	for key, event := range openByFingerprint {
		if _, hit := hits[key]; hit {
			continue
		}
		resolved := now
		if err := models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).
			Updates(map[string]any{"status": models.AlertStatusResolved, "resolved_at": &resolved}).Error; err != nil {
			log.Printf("证书到期告警：恢复事件 %d 失败: %v", event.ID, err)
			continue
		}
		if event.Status == models.AlertStatusFiring && event.LastNotifyAt != nil {
			e.notify(ctx, rule, event, true)
		}
	}
}

// fire 新建告警事件并投递通知
func (e *CertAlertEvaluator) fire(ctx context.Context, rule *models.OpsCertAlertRule, key string, hit certHit, now time.Time) {
	labels, _ := json.Marshal(buildCertLabels(hit.asset))
	instance := hit.asset.Ip
	if instance == "" {
		instance = hit.asset.Name
	}
	event := models.OpsMonitorAlertEvent{
		RuleId:      rule.ID,
		RuleName:    rule.Name,
		Fingerprint: key,
		InstanceIp:  truncate(instance, 32),
		Labels:      truncate(string(labels), 512),
		Value:       float64(hit.remaining),
		Threshold:   float64(rule.WarningDays),
		Severity:    hit.severity,
		Status:      models.AlertStatusFiring,
		FiredAt:     &now,
		Source:      models.AlertSourceCert,
		Summary:     buildCertSummary(rule, hit),
	}
	if err := models.DB.Create(&event).Error; err != nil {
		log.Printf("证书到期告警：创建事件失败 rule=%d: %v", rule.ID, err)
		return
	}

	if reason := certMuteReason(rule, now); reason != "" {
		models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).
			Update("suppressed", event.Suppressed+1)
		log.Printf("证书到期告警：事件 %d 已抑制（%s）", event.ID, reason)
		return
	}
	e.notify(ctx, rule, &event, false)
}

// renotify 已存在的事件随剩余天数变化更新级别/摘要，并按冷却重复提醒
func (e *CertAlertEvaluator) renotify(ctx context.Context, rule *models.OpsCertAlertRule, event *models.OpsMonitorAlertEvent, hit certHit, now time.Time) {
	summary := buildCertSummary(rule, hit)
	updates := map[string]any{
		"value": float64(hit.remaining), "severity": hit.severity,
		"summary": summary, "updated_at": now,
	}
	if event.Status != models.AlertStatusFiring {
		updates["status"] = models.AlertStatusFiring
		updates["fired_at"] = &now
	}
	if err := models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).Updates(updates).Error; err != nil {
		log.Printf("证书到期告警：更新事件 %d 失败: %v", event.ID, err)
		return
	}
	event.Value = float64(hit.remaining)
	event.Severity = hit.severity
	event.Summary = summary

	if event.Status == models.AlertStatusFiring && event.LastNotifyAt != nil &&
		now.Sub(*event.LastNotifyAt) < time.Duration(rule.CooldownSec)*time.Second {
		return
	}
	if reason := certMuteReason(rule, now); reason != "" {
		models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).
			Update("suppressed", event.Suppressed+1)
		return
	}
	e.notify(ctx, rule, event, false)
}

// notify 投递通知并记录通知时间与次数
func (e *CertAlertEvaluator) notify(ctx context.Context, rule *models.OpsCertAlertRule, event *models.OpsMonitorAlertEvent, recovered bool) {
	if strings.TrimSpace(rule.ChannelIds) == "" {
		return
	}
	success := e.notifier.Dispatch(ctx, event, rule.ChannelIds, recovered)
	if success == 0 {
		return
	}
	now := time.Now()
	updates := map[string]any{"notify_count": event.NotifyCount + success}
	if !recovered {
		updates["last_notify_at"] = &now
	}
	models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).Updates(updates)
}

// buildCertHit 计算资产的剩余天数与级别（已过期一律紧急）
func buildCertHit(rule *models.OpsCertAlertRule, asset models.OpsCmdbAsset, now time.Time) certHit {
	remaining := daysUntil(*asset.ExpiredAt)
	severity := rule.Severity
	if remaining < 0 {
		severity = models.SeverityCritical
	}
	return certHit{asset: asset, remaining: remaining, severity: severity}
}

func buildCertLabels(asset models.OpsCmdbAsset) map[string]string {
	return map[string]string{
		"__name__": "cert_expiry",
		"asset":    asset.Name,
		"model":    asset.ModelType,
		"owner":    asset.Owner,
	}
}

func buildCertSummary(rule *models.OpsCertAlertRule, hit certHit) string {
	expireAt := hit.asset.ExpiredAt.Format("2006-01-02 15:04")
	var state string
	switch {
	case hit.remaining < 0:
		state = fmt.Sprintf("已过期 %d 天", -hit.remaining)
	case hit.remaining == 0:
		state = "今天到期"
	default:
		state = fmt.Sprintf("剩余 %d 天", hit.remaining)
	}
	return truncate(fmt.Sprintf("%s「%s」到期时间 %s（%s，预警阈值 %d 天）",
		modelTypeText(hit.asset.ModelType), hit.asset.Name, expireAt, state, rule.WarningDays), 512)
}

// certMuteReason 返回抑制原因，空串表示允许通知
func certMuteReason(rule *models.OpsCertAlertRule, now time.Time) string {
	if rule.Muted == "1" {
		return "规则已静音"
	}
	if rule.SilenceUntil > 0 && time.Unix(rule.SilenceUntil, 0).After(now) {
		return "处于静默期"
	}
	if strings.TrimSpace(rule.ChannelIds) == "" {
		return "未配置通知渠道"
	}
	return ""
}

// certFingerprint 规则ID + 资产ID 的稳定指纹，保证同一资产在同一规则下只有一条未闭环事件
func certFingerprint(ruleId, assetId int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("cert|%d|%d", ruleId, assetId)))
	return hex.EncodeToString(sum[:])
}

// expiringAssets 查询预警窗口内（含已过期）且未退役的资产。
// 到期上界在 Go 侧算好再绑定，规避 MySQL DATE_ADD 与 PG 间隔语法的方言差异
func expiringAssets(modelTypes string, warningDays int, now time.Time) ([]models.OpsCmdbAsset, error) {
	upper := now.AddDate(0, 0, warningDays)
	var assets []models.OpsCmdbAsset
	db := models.DB.Where("del_flag = ? AND status <> ? AND expired_at IS NOT NULL AND expired_at <= ?",
		"0", models.CmdbStatusRetired, upper)
	if types := splitModelTypes(modelTypes); len(types) > 0 {
		db = db.Where("model_type IN ?", types)
	}
	err := db.Find(&assets).Error
	return assets, err
}

func splitModelTypes(raw string) []string {
	types := make([]string, 0, 4)
	for _, field := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			types = append(types, trimmed)
		}
	}
	return types
}

func modelTypeText(modelType string) string {
	switch modelType {
	case models.CmdbModelSsl:
		return "SSL证书"
	case models.CmdbModelDomain:
		return "域名"
	case models.CmdbModelServer:
		return "服务器"
	case models.CmdbModelDatabase:
		return "数据库"
	default:
		return modelType
	}
}
