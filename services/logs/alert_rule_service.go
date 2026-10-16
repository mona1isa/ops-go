package logs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/zhany/ops-go/controllers/logs/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/logs/datasource"
	"gorm.io/gorm"
)

// LogAlertService 日志告警规则管理
type LogAlertService struct{}

// List 规则分页
func (s *LogAlertService) List(pageNum, pageSize int, filter api.LogAlertPageRequest) (models.PageResult[models.OpsLogAlertRule], error) {
	result, err := models.Paginate[models.OpsLogAlertRule](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		db = db.Where("del_flag = ?", 0)
		if filter.Name != "" {
			db = db.Where("name like ?", "%"+escapeLike(filter.Name)+"%")
		}
		if filter.Level != "" {
			db = db.Where("level = ?", strings.ToUpper(filter.Level))
		}
		if filter.Muted != "" {
			db = db.Where("muted = ?", filter.Muted)
		}
		if filter.Enabled != "" {
			db = db.Where("enabled = ?", filter.Enabled)
		}
		return db.Order("id desc")
	})
	if err != nil {
		log.Println("查询日志告警规则失败：", err)
		return result, errors.New("查询日志告警规则失败")
	}
	return result, nil
}

// Get 规则详情
func (s *LogAlertService) Get(id int) (*models.OpsLogAlertRule, error) {
	var rule models.OpsLogAlertRule
	if err := models.DB.Where("id = ? AND del_flag = ?", id, 0).First(&rule).Error; err != nil {
		return nil, errors.New("告警规则不存在")
	}
	return &rule, nil
}

// Add 新增规则
func (s *LogAlertService) Add(rule *models.OpsLogAlertRule) error {
	if err := normalizeAlertRule(rule); err != nil {
		return err
	}
	if err := models.DB.Save(rule).Error; err != nil {
		log.Println("新增日志告警规则失败：", err)
		return errors.New("新增日志告警规则失败")
	}
	return nil
}

// Edit 更新规则
func (s *LogAlertService) Edit(request api.LogAlertRequest) error {
	var existing models.OpsLogAlertRule
	if err := models.DB.Where("id = ? AND del_flag = ?", request.Id, 0).First(&existing).Error; err != nil {
		return errors.New("告警规则不存在")
	}

	merged := existing
	merged.Name = request.Name
	merged.Selector = request.Selector
	merged.Host = request.Host
	merged.Source = request.Source
	merged.Keyword = request.Keyword
	merged.Level = request.Level
	merged.WindowSec = request.WindowSec
	merged.Threshold = request.Threshold
	merged.Severity = request.Severity
	merged.CooldownSec = request.CooldownSec
	merged.Muted = request.Muted
	merged.ChannelIds = request.ChannelIds
	merged.GroupByHost = request.GroupByHost
	merged.Enabled = request.Enabled
	merged.Remark = request.Remark
	merged.CreateBy = existing.CreateBy

	if err := normalizeAlertRule(&merged); err != nil {
		return err
	}

	if err := models.DB.Model(&models.OpsLogAlertRule{}).Where("id = ?", request.Id).
		Updates(map[string]any{
			"name": merged.Name, "selector": merged.Selector, "host": merged.Host,
			"source": merged.Source, "keyword": merged.Keyword, "level": merged.Level,
			"window_sec": merged.WindowSec, "threshold": merged.Threshold, "severity": merged.Severity,
			"cooldown_sec": merged.CooldownSec, "muted": merged.Muted, "channel_ids": merged.ChannelIds,
			"group_by_host": merged.GroupByHost, "enabled": merged.Enabled, "remark": merged.Remark,
			"create_by": merged.CreateBy, "updated_at": time.Now(),
		}).Error; err != nil {
		log.Println("更新日志告警规则失败：", err)
		return errors.New("更新日志告警规则失败")
	}
	return nil
}

// Delete 软删除规则，同时关闭未恢复的告警事件
func (s *LogAlertService) Delete(id int) error {
	return models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.OpsLogAlertRule{}).Where("id = ? AND del_flag = ?", id, 0).
			Update("del_flag", 1).Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&models.OpsMonitorAlertEvent{}).
			Where("rule_id = ? AND status <> ?", id, models.AlertStatusResolved).
			Updates(map[string]any{"status": models.AlertStatusResolved, "resolved_at": &now}).Error
	})
}

// Toggle 启停规则
func (s *LogAlertService) Toggle(id int, enabled string) error {
	if enabled != "0" && enabled != "1" {
		return errors.New("启停状态非法")
	}
	return models.DB.Model(&models.OpsLogAlertRule{}).Where("id = ? AND del_flag = ?", id, 0).
		Updates(map[string]any{"enabled": enabled, "updated_at": time.Now()}).Error
}

// Silence 设置静默截止时间，minute<=0 表示立即解除静默
func (s *LogAlertService) Silence(id int, minute int) error {
	until := int64(0)
	if minute > 0 {
		until = time.Now().Add(time.Duration(minute) * time.Minute).Unix()
	}
	return models.DB.Model(&models.OpsLogAlertRule{}).Where("id = ? AND del_flag = ?", id, 0).
		Updates(map[string]any{"silence_until": until, "updated_at": time.Now()}).Error
}

// Test 试算：用当前填写的规则条件立即查询一次窗口计数，不落库、不通知
func (s *LogAlertService) Test(ctx context.Context, request api.LogAlertRequest) (map[string]any, error) {
	rule := &models.OpsLogAlertRule{
		Name: request.Name, Selector: request.Selector, Host: request.Host, Source: request.Source,
		Keyword: request.Keyword, Level: request.Level, WindowSec: request.WindowSec,
		Threshold: request.Threshold, Severity: request.Severity, CooldownSec: request.CooldownSec,
		Muted: request.Muted, ChannelIds: request.ChannelIds, GroupByHost: request.GroupByHost,
		Enabled: request.Enabled,
	}
	if err := normalizeAlertRule(rule); err != nil {
		return nil, err
	}
	logql, err := BuildAlertLogQL(rule)
	if err != nil {
		return nil, err
	}
	samples, err := datasource.NewClient().QueryVector(ctx, logql, time.Now())
	if err != nil {
		return nil, err
	}
	hits := make([]datasource.VectorSample, 0, len(samples))
	for _, sample := range samples {
		if sample.Value >= float64(rule.Threshold) {
			hits = append(hits, sample)
		}
	}
	return map[string]any{
		"logql":   logql,
		"samples": samples,
		"hit":     len(hits) > 0,
	}, nil
}

// normalizeAlertRule 校验并补齐规则默认值
func normalizeAlertRule(rule *models.OpsLogAlertRule) error {
	rule.Name = strings.TrimSpace(rule.Name)
	if rule.Name == "" {
		return errors.New("规则名称不能为空")
	}
	if rule.Threshold <= 0 {
		rule.Threshold = 10
	}
	if rule.WindowSec <= 0 {
		rule.WindowSec = 300
	}
	if rule.WindowSec < 30 || rule.WindowSec > 3600 {
		return errors.New("统计时间窗需在 30~3600 秒之间")
	}
	if rule.CooldownSec < 0 {
		rule.CooldownSec = 0
	}
	if rule.CooldownSec == 0 {
		rule.CooldownSec = 300
	}
	if rule.Severity < 1 || rule.Severity > 3 {
		rule.Severity = models.SeverityWarning
	}
	rule.Level = strings.ToUpper(strings.TrimSpace(rule.Level))
	if rule.Level != "" && !isKnownLevel(rule.Level) {
		return errors.New("级别仅支持 ERROR/WARN/INFO/FATAL/DEBUG")
	}
	rule.Host = datasource.SafeLabelValue(rule.Host)
	rule.Source = datasource.SafeLabelValue(rule.Source)
	if len(rule.Keyword) > 255 {
		return errors.New("关键字过长")
	}
	if strings.ContainsAny(rule.Keyword, "\n\r") {
		return errors.New("关键字不能包含换行")
	}
	if rule.Selector != "" {
		if err := validateSelector(rule.Selector); err != nil {
			return err
		}
	}
	if rule.Muted != "1" {
		rule.Muted = "0"
	}
	if rule.GroupByHost != "0" {
		rule.GroupByHost = "1"
	}
	if rule.Enabled != "0" {
		rule.Enabled = "1"
	}
	// 至少要有一个过滤维度，否则等于全量日志计数，意义不大且开销高
	if rule.Selector == "" && rule.Host == "" && rule.Source == "" && rule.Level == "" && strings.TrimSpace(rule.Keyword) == "" {
		return errors.New("请至少设置主机、来源、级别或关键字之一")
	}
	return nil
}

func isKnownLevel(level string) bool {
	switch level {
	case "ERROR", "WARN", "INFO", "FATAL", "DEBUG", "TRACE":
		return true
	}
	return false
}

// BuildAlertLogQL 生成规则的计数表达式，供评估与界面试算共用
func BuildAlertLogQL(rule *models.OpsLogAlertRule) (string, error) {
	selector := strings.TrimSpace(rule.Selector)
	if selector == "" {
		labels := map[string]string{"job": "ops_log"}
		if rule.Host != "" {
			labels["host"] = rule.Host
		}
		if rule.Source != "" {
			labels["source"] = rule.Source
		}
		if rule.Level != "" {
			labels["level"] = rule.Level
		}
		selector = datasource.BuildSelector(labels)
	}
	if selector == "" {
		return "", errors.New("无法生成查询表达式：标签集合为空")
	}
	if err := validateSelector(selector); err != nil {
		return "", err
	}

	expr := selector
	if keyword := strings.TrimSpace(rule.Keyword); keyword != "" {
		if strings.ContainsAny(keyword, "\n\r") {
			return "", errors.New("关键字不能包含换行")
		}
		expr += ` |= "` + datasource.QuoteLabelValue(keyword) + `"`
	}

	window := fmt.Sprintf("%ds", rule.WindowSec)
	if rule.GroupByHost == "1" {
		return fmt.Sprintf("sum by (host) (count_over_time(%s [%s]))", expr, window), nil
	}
	return fmt.Sprintf("sum(count_over_time(%s [%s]))", expr, window), nil
}
