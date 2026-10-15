package monitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/monitor/datasource"
)

// EvaluatorService 告警规则评估服务：查询 Prometheus 并推进告警事件状态机
type EvaluatorService struct {
	Interval time.Duration
	client   *datasource.PrometheusClient
	notifier *NotifierService

	running   bool
	runningMu sync.Mutex
	ready     bool
	readyMu   sync.Mutex
	ruleErr   map[int]string
	ruleErrMu sync.Mutex
}

// NewEvaluatorService 按环境变量构造评估服务
func NewEvaluatorService() *EvaluatorService {
	return &EvaluatorService{
		Interval: time.Duration(evalIntervalSec()) * time.Second,
		client:   datasource.NewPrometheusClient(),
		notifier: &NotifierService{},
		ruleErr:  map[int]string{},
	}
}

// Start 常驻运行，启动前等待采集器把首批数据推入数据源
func (e *EvaluatorService) Start() {
	if e.Interval <= 0 {
		e.Interval = 30 * time.Second
	}
	log.Printf("告警规则评估服务已启动，评估间隔 %s", e.Interval)
	time.Sleep(5 * time.Second)
	e.evalOnce(context.Background())

	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()
	for range ticker.C {
		e.evalOnce(context.Background())
	}
}

// evalOnce 一轮评估：数据源不可用时整轮跳过，绝不把中断当作恢复
func (e *EvaluatorService) evalOnce(ctx context.Context) {
	if !e.acquireRound() {
		log.Println("上一轮告警评估尚未结束，跳过本轮")
		return
	}
	defer e.releaseRound()

	rules := e.loadRules()
	if len(rules) == 0 {
		return
	}
	if !e.client.Available(ctx) {
		e.logReady(false, len(rules))
		return
	}
	e.logReady(true, len(rules))

	for i := range rules {
		e.evalRule(ctx, &rules[i])
	}
}

func (e *EvaluatorService) acquireRound() bool {
	e.runningMu.Lock()
	defer e.runningMu.Unlock()
	if e.running {
		return false
	}
	e.running = true
	return true
}

func (e *EvaluatorService) releaseRound() {
	e.runningMu.Lock()
	e.running = false
	e.runningMu.Unlock()
}

func (e *EvaluatorService) loadRules() []models.OpsMonitorRule {
	var rules []models.OpsMonitorRule
	if err := models.DB.Where("del_flag = ? AND enabled = ?", 0, "1").Find(&rules).Error; err != nil {
		log.Println("查询启用中的告警规则异常:", err)
		return nil
	}
	return rules
}

// evalRule 单条规则的状态机推进：命中序列建/推进事件，消失的未闭环事件转恢复
func (e *EvaluatorService) evalRule(ctx context.Context, rule *models.OpsMonitorRule) {
	expr := strings.TrimSpace(rule.PromQL)
	if expr == "" {
		e.recordRuleError(rule.ID, "规则缺少 PromQL 表达式")
		return
	}
	samples, err := e.client.Query(ctx, expr)
	if err != nil {
		if errors.Is(err, datasource.ErrDatasourceUnavailable) {
			e.logReady(false, 1)
		} else {
			e.recordRuleError(rule.ID, err.Error())
		}
		return
	}
	e.recordRuleError(rule.ID, "")

	hits := map[string]datasource.Sample{}
	for _, sample := range samples {
		if math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0) {
			continue
		}
		if !hit(rule, sample.Value) {
			continue
		}
		hits[fingerprint(rule.ID, sample.Labels)] = sample
	}

	open := e.loadOpenEvents(rule.ID)
	now := time.Now()

	for fp, sample := range hits {
		event, exists := open[fp]
		if !exists {
			event = e.newEvent(rule, sample, fp, now)
			if err := models.DB.Create(event).Error; err != nil {
				log.Printf("创建告警事件失败 rule=%d: %v", rule.ID, err)
				continue
			}
			if rule.ForSec <= 0 {
				e.fire(ctx, rule, event, now)
			}
			continue
		}
		e.refresh(ctx, rule, event, sample, now)
		switch event.Status {
		case models.AlertStatusPending:
			if rule.ForSec <= 0 || elapsed(event.PendingSince, now) >= time.Duration(rule.ForSec)*time.Second {
				e.fire(ctx, rule, event, now)
			}
		case models.AlertStatusFiring:
			e.renotify(ctx, rule, event, now)
		}
	}

	for fp, event := range open {
		if _, ok := hits[fp]; ok {
			continue
		}
		e.resolve(ctx, rule, event, now)
	}
}

// hit 判断样本是否触发阈值。PromQL 自身已含比较时（阈值为 0）按“有结果即命中”处理。
func hit(rule *models.OpsMonitorRule, value float64) bool {
	if rule.Threshold == 0 {
		return true
	}
	return compare(value, compareOperator(rule), rule.Threshold)
}

func compare(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return value > threshold
	}
}

func compareOperator(rule *models.OpsMonitorRule) string {
	operator := strings.TrimSpace(rule.Operator)
	if operator == "" {
		return ">"
	}
	return operator
}

// fingerprint 规则ID + 标签集的稳定指纹，保证同一序列只有一条未闭环事件
func fingerprint(ruleId int, labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteString(strconv.Itoa(ruleId))
	for _, key := range keys {
		builder.WriteByte(0)
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(labels[key])
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func (e *EvaluatorService) loadOpenEvents(ruleId int) map[string]*models.OpsMonitorAlertEvent {
	var events []models.OpsMonitorAlertEvent
	if err := models.DB.Where("rule_id = ? AND status IN ?", ruleId,
		[]int8{models.AlertStatusPending, models.AlertStatusFiring}).Find(&events).Error; err != nil {
		log.Printf("查询未闭环告警事件失败 rule=%d: %v", ruleId, err)
		return map[string]*models.OpsMonitorAlertEvent{}
	}
	result := map[string]*models.OpsMonitorAlertEvent{}
	for i := range events {
		result[events[i].Fingerprint] = &events[i]
	}
	return result
}

func (e *EvaluatorService) newEvent(rule *models.OpsMonitorRule, sample datasource.Sample, fp string, now time.Time) *models.OpsMonitorAlertEvent {
	labels, _ := json.Marshal(sample.Labels)
	return &models.OpsMonitorAlertEvent{
		RuleId:       rule.ID,
		RuleName:     rule.Name,
		Fingerprint:  fp,
		InstanceIp:   sample.Instance(),
		Labels:       truncate(string(labels), 500),
		Value:        sample.Value,
		Threshold:    rule.Threshold,
		Severity:     rule.Severity,
		Status:       models.AlertStatusPending,
		PendingSince: &now,
		Summary:      buildSummary(rule, sample),
	}
}

// refresh 命中期间持续更新最新数值与摘要
func (e *EvaluatorService) refresh(ctx context.Context, rule *models.OpsMonitorRule, event *models.OpsMonitorAlertEvent, sample datasource.Sample, now time.Time) {
	summary := buildSummary(rule, sample)
	event.Value = sample.Value
	event.Summary = summary
	if err := models.DB.Model(event).Updates(map[string]interface{}{
		"value":      sample.Value,
		"summary":    summary,
		"updated_at": now,
	}).Error; err != nil {
		log.Printf("更新告警事件失败 event=%d: %v", event.ID, err)
	}
}

func (e *EvaluatorService) fire(ctx context.Context, rule *models.OpsMonitorRule, event *models.OpsMonitorAlertEvent, now time.Time) {
	event.Status = models.AlertStatusFiring
	event.FiredAt = &now
	if err := models.DB.Model(event).Updates(map[string]interface{}{
		"status":     models.AlertStatusFiring,
		"fired_at":   &now,
		"updated_at": now,
	}).Error; err != nil {
		log.Printf("告警事件转为触发失败 event=%d: %v", event.ID, err)
		return
	}
	e.notify(ctx, rule, event, now, false)
}

// renotify 持续告警按冷却时间重复通知
func (e *EvaluatorService) renotify(ctx context.Context, rule *models.OpsMonitorRule, event *models.OpsMonitorAlertEvent, now time.Time) {
	cooldown := time.Duration(rule.CooldownSec) * time.Second
	if event.LastNotifyAt != nil && now.Sub(*event.LastNotifyAt) < cooldown {
		e.markSuppressed(event, now)
		return
	}
	e.notify(ctx, rule, event, now, false)
}

func (e *EvaluatorService) resolve(ctx context.Context, rule *models.OpsMonitorRule, event *models.OpsMonitorAlertEvent, now time.Time) {
	fired := event.FiredAt != nil
	event.Status = models.AlertStatusResolved
	event.ResolvedAt = &now
	if err := models.DB.Model(event).Updates(map[string]interface{}{
		"status":      models.AlertStatusResolved,
		"resolved_at": &now,
		"updated_at":  now,
	}).Error; err != nil {
		log.Printf("告警事件转为恢复失败 event=%d: %v", event.ID, err)
		return
	}
	// 未触发过的事件（pending 直接消失）不产生恢复通知
	if !fired {
		return
	}
	e.notify(ctx, rule, event, now, true)
}

// notify 投递通知；静音与静默期只维护状态不发送，恢复通知绕过冷却
func (e *EvaluatorService) notify(ctx context.Context, rule *models.OpsMonitorRule, event *models.OpsMonitorAlertEvent, now time.Time, recovered bool) {
	if reason := muteReason(rule); reason != "" {
		e.markSuppressed(event, now)
		log.Printf("告警通知被抑制 event=%d rule=%d: %s", event.ID, rule.ID, reason)
		return
	}
	sent := e.notifier.Dispatch(ctx, event, rule.ChannelIds, recovered)
	event.NotifyCount += sent
	event.LastNotifyAt = &now
	updates := map[string]interface{}{"notify_count": event.NotifyCount, "updated_at": now}
	// 恢复通知不必占用冷却起点
	if !recovered {
		updates["last_notify_at"] = &now
	}
	if err := models.DB.Model(event).Updates(updates).Error; err != nil {
		log.Printf("记录告警通知结果失败 event=%d: %v", event.ID, err)
	}
}

func (e *EvaluatorService) markSuppressed(event *models.OpsMonitorAlertEvent, now time.Time) {
	event.Suppressed++
	if err := models.DB.Model(event).Updates(map[string]interface{}{
		"suppressed": event.Suppressed,
		"updated_at": now,
	}).Error; err != nil {
		log.Printf("记录告警抑制失败 event=%d: %v", event.ID, err)
	}
}

// muteReason 返回通知被静音/静默的原因，空串表示可通知
func muteReason(rule *models.OpsMonitorRule) string {
	if rule.Muted == "1" {
		return "规则已静音"
	}
	if rule.SilenceUntil > time.Now().Unix() {
		return "规则处于静默期"
	}
	if strings.TrimSpace(rule.ChannelIds) == "" {
		return "未配置通知渠道"
	}
	return ""
}

func buildSummary(rule *models.OpsMonitorRule, sample datasource.Sample) string {
	target := sample.Labels["instance"]
	if target == "" {
		target = "-"
	}
	if mount := sample.Labels["mountpoint"]; mount != "" {
		target = target + " " + mount
	}
	if device := sample.Labels["device"]; device != "" {
		target = target + " " + device
	}
	text := target + " 当前值 " + formatValue(sample.Value)
	if rule.Threshold != 0 {
		text += " " + compareOperator(rule) + " " + formatValue(rule.Threshold)
	}
	text = fmt.Sprintf("%s（%s）", text, rule.Name)
	return truncate(text, 500)
}

func formatValue(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

// elapsed 事件进入待确认至今的时长，缺失时间按已满足持续时长处理
func elapsed(since *time.Time, now time.Time) time.Duration {
	if since == nil {
		return time.Duration(math.MaxInt64)
	}
	return now.Sub(*since)
}

// recordRuleError 规则表达式错误只在首次或变化时打印，避免每轮刷屏
func (e *EvaluatorService) recordRuleError(ruleId int, msg string) {
	e.ruleErrMu.Lock()
	defer e.ruleErrMu.Unlock()
	previous, exists := e.ruleErr[ruleId]
	if msg == "" {
		delete(e.ruleErr, ruleId)
		if exists && previous != "" {
			log.Printf("告警规则 %d 已恢复正常评估", ruleId)
		}
		return
	}
	if exists && previous == msg {
		return
	}
	e.ruleErr[ruleId] = msg
	log.Printf("告警规则评估失败 rule=%d: %s", ruleId, msg)
}

// logReady 数据源可用性变化时打印一条日志
func (e *EvaluatorService) logReady(ready bool, ruleCount int) {
	e.readyMu.Lock()
	defer e.readyMu.Unlock()
	if e.ready == ready {
		return
	}
	e.ready = ready
	if ready {
		log.Printf("监控数据源已恢复，当前评估 %d 条告警规则", ruleCount)
		return
	}
	log.Println("监控数据源不可用，跳过本轮告警评估（不会把中断当作恢复）")
}
