package monitor

import (
	"errors"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// RuleService 告警规则管理服务
type RuleService struct{}

// RuleFilter 规则列表查询条件
type RuleFilter struct {
	Name     string
	Severity *int8
	Muted    string
	Enabled  string
}

// List 分页查询告警规则
func (s *RuleService) List(pageNum, pageSize int, filter RuleFilter) (models.PageResult[models.OpsMonitorRule], error) {
	result, err := models.Paginate[models.OpsMonitorRule](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		db = db.Where("del_flag = ?", "0")
		if filter.Name != "" {
			db = db.Where("name like ?", "%"+escapeLike(filter.Name)+"%")
		}
		if filter.Severity != nil {
			db = db.Where("severity = ?", *filter.Severity)
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
		log.Println("查询告警规则失败：", err)
		return result, errors.New("查询告警规则失败")
	}
	return result, nil
}

// Add 新增告警规则
func (s *RuleService) Add(rule *models.OpsMonitorRule) error {
	if err := validateRule(rule); err != nil {
		return err
	}
	if err := models.DB.Save(rule).Error; err != nil {
		log.Println("新增告警规则失败：", err)
		return errors.New("新增告警规则失败")
	}
	return nil
}

// Edit 编辑告警规则
func (s *RuleService) Edit(rule *models.OpsMonitorRule) error {
	var existing models.OpsMonitorRule
	if err := models.DB.Where("id = ? AND del_flag = ?", rule.ID, "0").First(&existing).Error; err != nil {
		return errors.New("告警规则不存在")
	}
	if err := validateRule(rule); err != nil {
		return err
	}
	rule.CreateBy = existing.CreateBy
	if err := models.DB.Model(&existing).Updates(map[string]interface{}{
		"name":          rule.Name,
		"metric":        rule.Metric,
		"prom_ql":       rule.PromQL,
		"operator":      rule.Operator,
		"threshold":     rule.Threshold,
		"severity":      rule.Severity,
		"for_sec":       rule.ForSec,
		"cooldown_sec":  rule.CooldownSec,
		"silence_until": rule.SilenceUntil,
		"muted":         rule.Muted,
		"channel_ids":   rule.ChannelIds,
		"enabled":       rule.Enabled,
		"remark":        rule.Remark,
		"update_by":     rule.UpdateBy,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		log.Println("编辑告警规则失败：", err)
		return errors.New("编辑告警规则失败")
	}
	return nil
}

// Delete 软删除告警规则，并闭环其未结束的事件
func (s *RuleService) Delete(id int) error {
	var existing models.OpsMonitorRule
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&existing).Error; err != nil {
		return errors.New("告警规则不存在")
	}
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&existing).Update("del_flag", "1").Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&models.OpsMonitorAlertEvent{}).
			Where("rule_id = ? AND status IN ?", id, []int8{models.AlertStatusPending, models.AlertStatusFiring}).
			Updates(map[string]interface{}{"status": models.AlertStatusResolved, "resolved_at": &now}).Error
	})
	if err != nil {
		log.Println("删除告警规则失败：", err)
		return errors.New("删除告警规则失败")
	}
	return nil
}

// Toggle 启用/停用告警规则
func (s *RuleService) Toggle(id int, enabled string) error {
	if enabled != "0" && enabled != "1" {
		return errors.New("启用状态非法")
	}
	if err := models.DB.Model(&models.OpsMonitorRule{}).Where("id = ? AND del_flag = ?", id, "0").
		Update("enabled", enabled).Error; err != nil {
		log.Println("更新告警规则状态失败：", err)
		return errors.New("更新告警规则状态失败")
	}
	return nil
}

// Silence 静默规则一段时间，秒数为 0 表示立即解除静默
func (s *RuleService) Silence(id int, seconds int64) error {
	if seconds < 0 {
		return errors.New("静默时长不能为负数")
	}
	until := int64(0)
	if seconds > 0 {
		until = time.Now().Unix() + seconds
	}
	if err := models.DB.Model(&models.OpsMonitorRule{}).Where("id = ? AND del_flag = ?", id, "0").
		Update("silence_until", until).Error; err != nil {
		log.Println("设置告警静默失败：", err)
		return errors.New("设置告警静默失败")
	}
	return nil
}

// TestResult 规则试算结果
type TestResult struct {
	Expression string    `json:"expression"`
	Matched    int       `json:"matched"`
	Hit        int       `json:"hit"`
	Values     []float64 `json:"values"`
	Message    string    `json:"message"`
}

// Test 用当前数据源即时试算表达式，不落库、不发通知
func (s *RuleService) Test(promQl, operator string, threshold float64) (*TestResult, error) {
	promQl = strings.TrimSpace(promQl)
	if promQl == "" {
		return nil, errors.New("请先填写查询表达式")
	}
	rule := &models.OpsMonitorRule{PromQL: promQl, Operator: operator, Threshold: threshold}
	if err := validateExpression(promQl); err != nil {
		return nil, err
	}

	ctx, cancel := queryContext()
	defer cancel()
	samples, err := NewMetricService().Query(ctx, promQl)
	if err != nil {
		return nil, err
	}
	result := &TestResult{Expression: promQl, Matched: len(samples)}
	for _, sample := range samples {
		if math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0) {
			continue
		}
		if len(result.Values) < 20 {
			result.Values = append(result.Values, sample.Value)
		}
		if hit(rule, sample.Value) {
			result.Hit++
		}
	}
	switch {
	case result.Matched == 0:
		result.Message = "表达式没有匹配到任何监控序列"
	case result.Hit == 0:
		result.Message = "表达式有效，当前没有序列满足告警条件"
	default:
		result.Message = "表达式有效，当前有 " + strconv.Itoa(result.Hit) + " 条序列满足告警条件"
	}
	return result, nil
}

func validateRule(rule *models.OpsMonitorRule) error {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.PromQL = strings.TrimSpace(rule.PromQL)
	if rule.Name == "" {
		return errors.New("规则名称不能为空")
	}
	if len([]rune(rule.Name)) > 128 {
		return errors.New("规则名称过长")
	}
	if err := validateExpression(rule.PromQL); err != nil {
		return err
	}
	if rule.Threshold != 0 || strings.TrimSpace(rule.Operator) != "" {
		operator := compareOperator(rule)
		switch operator {
		case ">", ">=", "<", "<=", "==", "!=":
			rule.Operator = operator
		default:
			return errors.New("比较符仅支持 > >= < <= == !=")
		}
	} else {
		rule.Operator = ">"
	}
	switch rule.Severity {
	case models.SeverityCritical, models.SeverityWarning, models.SeverityInfo:
	default:
		rule.Severity = models.SeverityWarning
	}
	if rule.ForSec < 0 || rule.ForSec > 86400 {
		return errors.New("持续时长需在 0 到 86400 秒之间")
	}
	if rule.CooldownSec < 0 {
		rule.CooldownSec = 0
	}
	if rule.CooldownSec > 86400*30 {
		return errors.New("冷却时长不能超过 30 天")
	}
	if rule.SilenceUntil < 0 {
		rule.SilenceUntil = 0
	}
	if rule.Muted != "0" && rule.Muted != "1" {
		rule.Muted = "0"
	}
	if rule.Enabled != "0" && rule.Enabled != "1" {
		rule.Enabled = "1"
	}
	rule.ChannelIds = channelIdsText(rule.ChannelIds)
	rule.Metric = truncate(strings.TrimSpace(rule.Metric), 64)
	rule.PromQL = truncate(rule.PromQL, 512)
	return nil
}

// validateExpression 拦截明显非法或危险的表达式：只读查询，禁止写类 API 语句
func validateExpression(promQl string) error {
	if promQl == "" {
		return errors.New("查询表达式不能为空")
	}
	if len(promQl) > 512 {
		return errors.New("查询表达式不能超过 512 个字符")
	}
	if strings.ContainsAny(promQl, "\n\r\x00") {
		return errors.New("查询表达式不能包含换行")
	}
	if strings.Contains(promQl, ";") {
		return errors.New("查询表达式不能包含分号")
	}
	if err := checkParentheses(promQl); err != nil {
		return err
	}
	return nil
}

// checkParentheses 括号与引号配对检查，避免把明显写坏的表达式送进数据源
func checkParentheses(expr string) error {
	depth := 0
	inString := false
	escaped := false
	for _, char := range expr {
		if escaped {
			escaped = false
			continue
		}
		switch char {
		case '\\':
			if inString {
				escaped = true
			}
		case '"', '\'':
			inString = !inString
		case '(':
			if !inString {
				depth++
			}
		case ')':
			if !inString {
				depth--
			}
			if depth < 0 {
				return errors.New("查询表达式括号不匹配")
			}
		}
	}
	if inString {
		return errors.New("查询表达式引号不匹配")
	}
	if depth != 0 {
		return errors.New("查询表达式括号不匹配")
	}
	return nil
}

// escapeLike 转义 LIKE 通配符，避免用户输入的 % 和 _ 影响匹配范围
func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
