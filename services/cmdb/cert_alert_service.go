package cmdb

import (
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// CertAlertService 证书/域名到期告警规则管理 + 到期概览查询
type CertAlertService struct{}

// CertRulePageRequest 规则分页查询条件
type CertRulePageRequest struct {
	Name    string
	Enabled string
	Muted   string
}

// CertAlertRow 到期概览行：资产 + 计算出的剩余天数
type CertAlertRow struct {
	models.OpsCmdbAsset
	Remaining int    `json:"remaining"` // 剩余天数，负数表示已过期天数
	Level     int8   `json:"level"`     // 派生级别（1 已过期/紧急 2 临期警告 3 关注）
	LevelText string `json:"levelText"`
}

// ListRules 规则分页
func (s *CertAlertService) ListRules(pageNum, pageSize int, filter CertRulePageRequest) (models.PageResult[models.OpsCertAlertRule], error) {
	result, err := models.Paginate[models.OpsCertAlertRule](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		db = db.Where("del_flag = ?", "0")
		if filter.Name != "" {
			db = db.Where("name like ?", "%"+escapeLike(filter.Name)+"%")
		}
		if filter.Enabled != "" {
			db = db.Where("enabled = ?", filter.Enabled)
		}
		if filter.Muted != "" {
			db = db.Where("muted = ?", filter.Muted)
		}
		return db.Order("id desc")
	})
	if err != nil {
		log.Println("查询证书告警规则失败：", err)
		return result, errors.New("查询证书告警规则失败")
	}
	return result, nil
}

// AddRule 新增规则
func (s *CertAlertService) AddRule(rule *models.OpsCertAlertRule) error {
	if err := normalizeCertRule(rule); err != nil {
		return err
	}
	if err := models.DB.Save(rule).Error; err != nil {
		log.Println("新增证书告警规则失败：", err)
		return errors.New("新增证书告警规则失败")
	}
	return nil
}

// EditRule 更新规则
func (s *CertAlertService) EditRule(rule *models.OpsCertAlertRule) error {
	var existing models.OpsCertAlertRule
	if err := models.DB.Where("id = ? AND del_flag = ?", rule.ID, "0").First(&existing).Error; err != nil {
		return errors.New("告警规则不存在")
	}
	if err := normalizeCertRule(rule); err != nil {
		return err
	}
	if err := models.DB.Model(&models.OpsCertAlertRule{}).Where("id = ?", rule.ID).
		Updates(map[string]any{
			"name": rule.Name, "model_types": rule.ModelTypes, "warning_days": rule.WarningDays,
			"severity": rule.Severity, "cooldown_sec": rule.CooldownSec, "muted": rule.Muted,
			"channel_ids": rule.ChannelIds, "enabled": rule.Enabled, "remark": rule.Remark,
			"create_by": existing.CreateBy, "updated_at": time.Now(),
		}).Error; err != nil {
		log.Println("更新证书告警规则失败：", err)
		return errors.New("更新证书告警规则失败")
	}
	return nil
}

// DeleteRule 软删除规则，同时关闭该规则未恢复的告警事件
func (s *CertAlertService) DeleteRule(id int) error {
	return models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.OpsCertAlertRule{}).Where("id = ? AND del_flag = ?", id, "0").
			Update("del_flag", "1").Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&models.OpsMonitorAlertEvent{}).
			Where("rule_id = ? AND source = ? AND status <> ?", id, models.AlertSourceCert, models.AlertStatusResolved).
			Updates(map[string]any{"status": models.AlertStatusResolved, "resolved_at": &now}).Error
	})
}

// ToggleRule 启停规则
func (s *CertAlertService) ToggleRule(id int, enabled string) error {
	if enabled != "0" && enabled != "1" {
		return errors.New("启停状态非法")
	}
	return models.DB.Model(&models.OpsCertAlertRule{}).Where("id = ? AND del_flag = ?", id, "0").
		Updates(map[string]any{"enabled": enabled, "updated_at": time.Now()}).Error
}

// SilenceRule 设置静默截止时间，minute<=0 表示立即解除静默
func (s *CertAlertService) SilenceRule(id int, minute int) error {
	until := int64(0)
	if minute > 0 {
		until = time.Now().Add(time.Duration(minute) * time.Minute).Unix()
	}
	return models.DB.Model(&models.OpsCertAlertRule{}).Where("id = ? AND del_flag = ?", id, "0").
		Updates(map[string]any{"silence_until": until, "updated_at": time.Now()}).Error
}

// normalizeCertRule 校验并补齐规则默认值
func normalizeCertRule(rule *models.OpsCertAlertRule) error {
	rule.Name = strings.TrimSpace(rule.Name)
	if rule.Name == "" {
		return errors.New("规则名称不能为空")
	}
	if rule.WarningDays <= 0 {
		rule.WarningDays = 30
	}
	if rule.WarningDays > 3650 {
		return errors.New("预警天数需在 1~3650 之间")
	}
	if rule.CooldownSec <= 0 {
		rule.CooldownSec = 86400
	}
	if rule.Severity < models.SeverityCritical || rule.Severity > models.SeverityInfo {
		rule.Severity = models.SeverityWarning
	}
	// 过滤非法模型类型，只保留已知类型；全空表示监控所有含到期时间的资产
	valid := make([]string, 0, len(rule.ModelTypes))
	for _, t := range splitModelTypes(rule.ModelTypes) {
		for _, known := range models.CmdbModelTypes {
			if t == known {
				valid = append(valid, t)
				break
			}
		}
	}
	rule.ModelTypes = strings.Join(valid, ",")
	if rule.Muted != "1" {
		rule.Muted = "0"
	}
	if rule.Enabled != "0" {
		rule.Enabled = "1"
	}
	return nil
}

// ExpiringAssets 到期概览：返回预警窗口内（含已过期）且未退役的资产，按到期时间升序。
// warningDays<=0 时默认取 30 天，modelTypes 为空表示所有含到期时间的资产
func (s *CertAlertService) ExpiringAssets(modelTypes string, warningDays int) ([]CertAlertRow, error) {
	if warningDays <= 0 {
		warningDays = 30
	}
	rows, err := expiringAssets(modelTypes, warningDays, time.Now())
	if err != nil {
		log.Println("查询到期资产失败：", err)
		return nil, errors.New("查询到期资产失败")
	}
	out := make([]CertAlertRow, 0, len(rows))
	for _, asset := range rows {
		if asset.ExpiredAt == nil {
			continue
		}
		out = append(out, toCertRow(asset))
	}
	return out, nil
}

// toCertRow 依据到期时间派生剩余天数与关注级别
func toCertRow(asset models.OpsCmdbAsset) CertAlertRow {
	remaining := daysUntil(*asset.ExpiredAt)
	level := int8(models.SeverityInfo)
	levelText := "关注"
	switch {
	case remaining < 0:
		level = models.SeverityCritical
		levelText = "已过期"
	case remaining <= 7:
		level = models.SeverityWarning
		levelText = "临期"
	}
	return CertAlertRow{OpsCmdbAsset: asset, Remaining: remaining, Level: level, LevelText: levelText}
}

// daysUntil 距到期的整天数：按日历天（当日零点与到期日零点之差）计算，
// 避免时钟小时 floor 因执行时刻微秒漂移造成 ±1 天误差；负数表示已过期天数。
// 零点差可能因夏令时为 23/25 小时，故用四舍五入而非截断
func daysUntil(expireAt time.Time) int {
	now := time.Now()
	local := expireAt.Local()
	day0 := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day1 := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	return int(math.Round(day1.Sub(day0).Hours() / 24))
}
