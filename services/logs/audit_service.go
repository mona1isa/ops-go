package logs

import (
	"errors"
	"log"
	"time"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// SysLogFilter 操作审计查询条件，空值表示不过滤
type SysLogFilter struct {
	CreateUser string
	OperModule string
	OperType   string
	Sensitive  string
	RequestUri string
	StatusCode string
	BizFail    string // "1" 只看业务失败
	StartAt    string
	EndAt      string
}

// PageSysLog 操作审计日志分页
func PageSysLog(pageNum, pageSize int, filter SysLogFilter) (models.PageResult[models.SysLog], error) {
	result, err := models.Paginate[models.SysLog](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		if filter.CreateUser != "" {
			db = db.Where("create_user = ?", filter.CreateUser)
		}
		if filter.OperModule != "" {
			db = db.Where("oper_module = ?", filter.OperModule)
		}
		if filter.OperType != "" {
			db = db.Where("oper_type = ?", filter.OperType)
		}
		if filter.Sensitive == "1" {
			db = db.Where(models.QuotedIdent("sensitive")+" = ?", "1")
		}
		if filter.RequestUri != "" {
			db = db.Where("request_uri like ?", "%"+escapeLike(filter.RequestUri)+"%")
		}
		if filter.StatusCode != "" {
			db = db.Where("status_code = ?", filter.StatusCode)
		}
		if filter.BizFail == "1" {
			db = db.Where("biz_code <> '' AND biz_code <> ?", "200")
		}
		if filter.StartAt != "" {
			db = db.Where("created_at >= ?", parseTimeInput(filter.StartAt))
		}
		if filter.EndAt != "" {
			db = db.Where("created_at <= ?", parseTimeInput(filter.EndAt))
		}
		return db.Where("del_flag = ?", "0").Order("id desc")
	})
	if err != nil {
		log.Println("查询操作审计日志失败：", err)
		return result, errors.New("查询操作审计日志失败")
	}
	return result, nil
}

// AuditOverview 操作行为分析：模块分布、活跃操作人、敏感操作留痕、异常与慢接口
type AuditOverview struct {
	Hours        int64     `json:"hours"`
	Total        int64     `json:"total"`
	Fail         int64     `json:"fail"`
	Sensitive    int64     `json:"sensitive"`
	Modules      []TopItem `json:"modules"`
	Operators    []TopItem `json:"operators"`
	SensitiveOps []TopItem `json:"sensitiveOps"`
	SlowApis     []SlowApi `json:"slowApis"`
}

// SlowApi 慢接口统计
type SlowApi struct {
	RequestUri string  `json:"requestUri"`
	Cnt        int64   `json:"cnt"`
	AvgCostMs  float64 `json:"avgCostMs"`
	MaxCostMs  int64   `json:"maxCostMs"`
}

// AnalyzeAudit 统计最近 hours 小时的操作行为，默认 24 小时，最长 720 小时（30 天）
func AnalyzeAudit(hours int64) AuditOverview {
	if hours <= 0 {
		hours = 24
	}
	if hours > 720 {
		hours = 720
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour).Format("2006-01-02 15:04:05")
	overview := AuditOverview{
		Hours:        hours,
		Modules:      []TopItem{},
		Operators:    []TopItem{},
		SensitiveOps: []TopItem{},
		SlowApis:     []SlowApi{},
	}

	var total int64
	models.DB.Model(&models.SysLog{}).Where("created_at >= ?", since).Count(&total)
	overview.Total = total
	models.DB.Model(&models.SysLog{}).Where("created_at >= ? AND "+models.QuotedIdent("sensitive")+" = ?", since, "1").Count(&overview.Sensitive)
	models.DB.Model(&models.SysLog{}).Where("created_at >= ? AND (status_code >= ? OR (biz_code <> '' AND biz_code <> ?))", since, "400", "200").Count(&overview.Fail)

	models.DB.Table("sys_log").
		Select("oper_module as name, COUNT(*) as count").
		Where("created_at >= ? AND del_flag = ?", since, "0").
		Group("oper_module").Order("count desc").Limit(20).Scan(&overview.Modules)

	models.DB.Table("sys_log").
		Select("create_user as name, COUNT(*) as count").
		Where("created_at >= ? AND del_flag = ? AND create_user <> ''", since, "0").
		Group("create_user").Order("count desc").Limit(10).Scan(&overview.Operators)

	models.DB.Table("sys_log").
		Select("request_uri as name, COUNT(*) as count").
		Where("created_at >= ? AND del_flag = ? AND "+models.QuotedIdent("sensitive")+" = ?", since, "0", "1").
		Group("request_uri").Order("count desc").Limit(20).Scan(&overview.SensitiveOps)

	models.DB.Table("sys_log").
		Select("request_uri as request_uri, COUNT(*) as cnt, AVG(cost_time_ms) as avg_cost_ms, MAX(cost_time_ms) as max_cost_ms").
		Where("created_at >= ? AND del_flag = ? AND cost_time_ms > 0", since, "0").
		Group("request_uri").Order("avg_cost_ms desc").Limit(10).Scan(&overview.SlowApis)

	overview.Modules = nilToEmpty(overview.Modules)
	overview.Operators = nilToEmpty(overview.Operators)
	overview.SensitiveOps = nilToEmpty(overview.SensitiveOps)
	overview.SlowApis = nilToEmptyApis(overview.SlowApis)
	return overview
}

func nilToEmptyApis(items []SlowApi) []SlowApi {
	if items == nil {
		return []SlowApi{}
	}
	return items
}
