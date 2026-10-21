package logs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/logs/datasource"
	"github.com/zhany/ops-go/services/monitor"
)

// LogAlertEvaluator 日志告警评估服务：按时间窗统计命中条数，超阈值复用监控告警链路投递
type LogAlertEvaluator struct {
	client  *datasource.Client
	notify  monitor.NotifierService
	running atomic.Bool
}

// NewLogAlertEvaluator 创建日志告警评估服务
func NewLogAlertEvaluator() *LogAlertEvaluator {
	return &LogAlertEvaluator{client: datasource.NewClient()}
}

// Start 启动评估定时任务
func (e *LogAlertEvaluator) Start() {
	interval := time.Duration(alertIntervalSec()) * time.Second
	log.Printf("日志告警评估服务已启动，轮询间隔 %v，Loki: %s", interval, e.client.BaseURL())

	time.Sleep(8 * time.Second)
	e.evaluateOnce()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		e.evaluateOnce()
	}
}

func alertIntervalSec() int {
	return datasource.EnvInt("LOG_ALERT_INTERVAL", 60)
}

// evaluateOnce 单轮评估：逐条规则查询窗口计数并推进事件状态
func (e *LogAlertEvaluator) evaluateOnce() {
	if e.running.Swap(true) {
		return
	}
	defer e.running.Store(false)

	var rules []models.OpsLogAlertRule
	if err := models.DB.Where("del_flag = ? AND enabled = ?", "0", "1").Find(&rules).Error; err != nil {
		log.Println("日志告警：查询规则失败:", err)
		return
	}
	if len(rules) == 0 {
		return
	}
	if !e.client.Available(context.Background()) {
		log.Println("日志告警：Loki 不可达，跳过本轮")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(rules)*10+30)*time.Second)
	defer cancel()

	for i := range rules {
		e.evaluateRule(ctx, &rules[i])
	}
}

// evaluateRule 评估单条规则，返回本轮新增/恢复的事件数便于日志观测
func (e *LogAlertEvaluator) evaluateRule(ctx context.Context, rule *models.OpsLogAlertRule) {
	logql, err := BuildAlertLogQL(rule)
	if err != nil {
		log.Printf("日志告警：规则 %d 表达式非法: %v", rule.ID, err)
		return
	}

	samples, err := e.client.QueryVector(ctx, logql, time.Now())
	if err != nil {
		log.Printf("日志告警：规则 %d 查询失败: %v", rule.ID, err)
		return
	}

	hits := make(map[string]datasource.VectorSample, len(samples))
	for _, sample := range samples {
		if sample.Value < float64(rule.Threshold) {
			continue
		}
		hits[fingerprint(rule.ID, sample.Labels)] = sample
	}

	var open []models.OpsMonitorAlertEvent
	if err := models.DB.Where("rule_id = ? AND source = ? AND status IN ?",
		rule.ID, models.AlertSourceLog, []int{models.AlertStatusPending, models.AlertStatusFiring}).
		Find(&open).Error; err != nil {
		log.Printf("日志告警：规则 %d 读取未恢复事件失败: %v", rule.ID, err)
		return
	}

	openByFingerprint := make(map[string]*models.OpsMonitorAlertEvent, len(open))
	for i := range open {
		openByFingerprint[open[i].Fingerprint] = &open[i]
	}

	now := time.Now()
	for key, sample := range hits {
		if event, exists := openByFingerprint[key]; exists {
			e.renotify(ctx, rule, event, sample.Value, now)
			continue
		}
		e.fire(ctx, rule, key, sample, now)
	}

	// 窗口内不再命中：关闭历史事件，便于下一次命中重新产生告警
	for key, event := range openByFingerprint {
		if _, hit := hits[key]; hit {
			continue
		}
		resolved := now
		if err := models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).
			Updates(map[string]any{"status": models.AlertStatusResolved, "resolved_at": &resolved}).Error; err != nil {
			log.Printf("日志告警：恢复事件 %d 失败: %v", event.ID, err)
			continue
		}
		// 恢复通知只在真正告警过之后发送，避免抖动刷屏
		if event.Status == models.AlertStatusFiring && event.LastNotifyAt != nil {
			e.notify.Dispatch(ctx, event, rule.ChannelIds, true)
		}
	}
}

// fire 新建告警事件并投递通知
func (e *LogAlertEvaluator) fire(ctx context.Context, rule *models.OpsLogAlertRule, key string, sample datasource.VectorSample, now time.Time) {
	labels, _ := json.Marshal(sample.Labels)
	event := models.OpsMonitorAlertEvent{
		RuleId:      rule.ID,
		RuleName:    rule.Name,
		Fingerprint: key,
		InstanceIp:  sample.Labels["host"],
		Labels:      truncate(string(labels), 512),
		Value:       sample.Value,
		Threshold:   float64(rule.Threshold),
		Severity:    rule.Severity,
		Status:      models.AlertStatusFiring,
		FiredAt:     &now,
		Source:      models.AlertSourceLog,
		Summary:     buildSummary(rule, sample),
	}
	if err := models.DB.Create(&event).Error; err != nil {
		log.Printf("日志告警：创建事件失败 rule=%d: %v", rule.ID, err)
		return
	}

	if reason := muteReason(rule, now); reason != "" {
		models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).
			Update("suppressed", event.Suppressed+1)
		log.Printf("日志告警：事件 %d 已抑制（%s）", event.ID, reason)
		return
	}

	e.dispatch(ctx, rule, &event, false)
}

// renotify 已存在的告警按冷却时间重复提醒
func (e *LogAlertEvaluator) renotify(ctx context.Context, rule *models.OpsLogAlertRule, event *models.OpsMonitorAlertEvent, value float64, now time.Time) {
	updates := map[string]any{"value": value, "updated_at": now}
	if event.Status != models.AlertStatusFiring {
		updates["status"] = models.AlertStatusFiring
		updates["fired_at"] = &now
	}
	if err := models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).Updates(updates).Error; err != nil {
		log.Printf("日志告警：更新事件 %d 失败: %v", event.ID, err)
		return
	}
	event.Value = value

	if event.Status == models.AlertStatusFiring && event.LastNotifyAt != nil &&
		now.Sub(*event.LastNotifyAt) < time.Duration(rule.CooldownSec)*time.Second {
		return
	}
	if reason := muteReason(rule, now); reason != "" {
		models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).
			Update("suppressed", event.Suppressed+1)
		return
	}
	e.dispatch(ctx, rule, event, false)
}

// dispatch 投递通知并记录通知时间与次数
func (e *LogAlertEvaluator) dispatch(ctx context.Context, rule *models.OpsLogAlertRule, event *models.OpsMonitorAlertEvent, recovered bool) {
	if strings.TrimSpace(rule.ChannelIds) == "" {
		return
	}
	success := e.notify.Dispatch(ctx, event, rule.ChannelIds, recovered)
	if success == 0 {
		return
	}
	now := time.Now()
	models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("id = ?", event.ID).
		Updates(map[string]any{"last_notify_at": &now, "notify_count": event.NotifyCount + success})
}

// buildSummary 告警摘要，直接可读地说明「哪台主机、多少条、什么条件」
func buildSummary(rule *models.OpsLogAlertRule, sample datasource.VectorSample) string {
	host := sample.Labels["host"]
	if host == "" {
		host = "全部主机"
	}
	condition := rule.Name
	if rule.Level != "" {
		condition = fmt.Sprintf("%s 级别 %s", condition, rule.Level)
	}
	if rule.Keyword != "" {
		condition = fmt.Sprintf("%s 关键字 %q", condition, rule.Keyword)
	}
	return truncate(fmt.Sprintf("%d 秒内命中 %d 条（阈值 %d）：%s（主机 %s）",
		rule.WindowSec, int(sample.Value), rule.Threshold, condition, host), 512)
}

// muteReason 返回抑制原因，空串表示允许通知
func muteReason(rule *models.OpsLogAlertRule, now time.Time) string {
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

// fingerprint 规则ID + 标签集的稳定指纹，保证同一条序列只有一个未恢复事件
func fingerprint(ruleId int, labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	fmt.Fprintf(&builder, "log|%d|", ruleId)
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(labels[key])
		builder.WriteByte(';')
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}
