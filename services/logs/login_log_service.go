package logs

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/utils"
	"gorm.io/gorm"
)

// LoginLogEntry 登录事件，由登录/登出链路构造
type LoginLogEntry struct {
	UserName  string
	UserId    int
	DeptId    int
	Result    int8
	Reason    string
	Ip        string
	UserAgent string
	Platform  string
	// Password 仅在密码校验失败时传入，用于计算不可还原的输入特征。
	// 本字段不入库：进入 RecordLogin 即被消费并清零，调用方不应保留副本。
	// 传登录口令还是动态码/恢复码均可，特征口径一致（见 DescribeCredential）。
	Password string
}

// RecordLogin 异步落库登录日志。
// 归属地查询和写库都不允许阻塞登录主链路，失败只打日志。
func RecordLogin(entry LoginLogEntry) {
	if entry.UserName == "" && entry.Reason == "" {
		return
	}
	// 在起协程前就地算出特征并抹掉明文，保证口令生命周期止于本函数栈
	pwdLength, pwdFlags := DescribeCredential(entry.Password)
	entry.Password = ""
	snapshot := entry
	go func() {
		record := models.SysLoginLog{
			UserName:  snapshot.UserName,
			UserId:    snapshot.UserId,
			DeptId:    snapshot.DeptId,
			Result:    snapshot.Result,
			Reason:    truncate(snapshot.Reason, 255),
			PwdLength: pwdLength,
			PwdFlags:  truncate(pwdFlags, 128),
			IpAddr:    snapshot.Ip,
			UserAgent: truncate(snapshot.UserAgent, 512),
			Browser:   parseBrowser(snapshot.UserAgent),
			Os:        parseOs(snapshot.UserAgent),
			Platform:  snapshot.Platform,
			CreatedAt: time.Now(),
		}
		if record.Platform == "" {
			record.Platform = models.LoginSourceWeb
		}
		if region, err := utils.GetRegion(snapshot.Ip); err == nil {
			record.Location = truncate(region, 128)
		}
		if err := models.DB.Create(&record).Error; err != nil {
			log.Printf("写入登录日志失败 user=%s: %v", snapshot.UserName, err)
		}
	}()
}

// LoginLogFilter 登录日志查询条件，空值表示不过滤
type LoginLogFilter struct {
	UserName string
	IpAddr   string
	Result   *int8
	Platform string
	StartAt  string
	EndAt    string
}

// PageLoginLog 登录日志分页
func PageLoginLog(pageNum, pageSize int, filter LoginLogFilter) (models.PageResult[models.SysLoginLog], error) {
	result, err := models.Paginate[models.SysLoginLog](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		if filter.UserName != "" {
			db = db.Where("user_name like ?", "%"+escapeLike(filter.UserName)+"%")
		}
		if filter.IpAddr != "" {
			db = db.Where("ip_addr like ?", "%"+escapeLike(filter.IpAddr)+"%")
		}
		if filter.Result != nil {
			db = db.Where("result = ?", *filter.Result)
		}
		if filter.Platform != "" {
			db = db.Where("platform = ?", filter.Platform)
		}
		if filter.StartAt != "" {
			db = db.Where("created_at >= ?", parseTimeInput(filter.StartAt))
		}
		if filter.EndAt != "" {
			db = db.Where("created_at <= ?", parseTimeInput(filter.EndAt))
		}
		return db.Order("id desc")
	})
	if err != nil {
		log.Println("查询登录日志失败：", err)
		return result, errors.New("查询登录日志失败")
	}
	return result, nil
}

// LoginTrendPoint 登录趋势单日点位
type LoginTrendPoint struct {
	Date    string `json:"date"`
	Success int64  `json:"success"`
	Fail    int64  `json:"fail"`
}

// TopItem 通用「按维度计数」结果
type TopItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// LoginOverview 登录行为分析：成功率趋势、失败来源 Top、活跃账号 Top、可疑账号
type LoginOverview struct {
	Days        int               `json:"days"`
	Total       int64             `json:"total"`
	Success     int64             `json:"success"`
	Fail        int64             `json:"fail"`
	Pending     int64             `json:"pending"`
	SuccessRate float64           `json:"successRate"`
	Trend       []LoginTrendPoint `json:"trend"`
	FailUsers   []TopItem         `json:"failUsers"`
	FailIps     []TopItem         `json:"failIps"`
	ActiveUsers []TopItem         `json:"activeUsers"`
	Suspicious  []TopItem         `json:"suspicious"`
}

// AnalyzeLogin 统计最近 days 天的登录行为。
// 可疑账号口径：失败次数 >= 5，用于快速定位爆破与异常登录。
// 中间态（等待 MFA 验证/引导绑定）既不是成功也不是失败，登出也不是登录尝试，
// 因此总数与成功率只统计有明确结果的两类，否则启用 MFA 的账号会被误判为可疑。
func AnalyzeLogin(days int) LoginOverview {
	if days <= 0 || days > 90 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	overview := LoginOverview{Days: days, Trend: []LoginTrendPoint{}, FailUsers: []TopItem{}, FailIps: []TopItem{}, ActiveUsers: []TopItem{}, Suspicious: []TopItem{}}

	models.DB.Model(&models.SysLoginLog{}).Where("created_at >= ? AND result = ?", since, models.LoginResultSuccess).Count(&overview.Success)
	models.DB.Model(&models.SysLoginLog{}).Where("created_at >= ? AND result = ?", since, models.LoginResultFail).Count(&overview.Fail)
	models.DB.Model(&models.SysLoginLog{}).Where("created_at >= ? AND result = ?", since, models.LoginResultPending).Count(&overview.Pending)
	overview.Total = overview.Success + overview.Fail
	if overview.Total > 0 {
		overview.SuccessRate = float64(overview.Success) / float64(overview.Total) * 100
	}

	// 按天分组：MySQL 用 DATE_FORMAT，PostgreSQL 用 to_char（models.DayExpr）
	type dayRow struct {
		Date    string
		Success int64
		Fail    int64
	}
	var rows []dayRow
	dayExpr := models.DayExpr("created_at")
	models.DB.Table("sys_login_log").
		Select(dayExpr+" as date, "+
			"SUM(CASE WHEN result = 1 THEN 1 ELSE 0 END) as success, "+
			"SUM(CASE WHEN result = 0 THEN 1 ELSE 0 END) as fail").
		Where("created_at >= ?", since).
		Group(dayExpr).
		Order("date asc").Scan(&rows)
	for _, row := range rows {
		overview.Trend = append(overview.Trend, LoginTrendPoint{Date: row.Date, Success: row.Success, Fail: row.Fail})
	}

	overview.FailUsers = topUsersByResult(since, models.LoginResultFail)
	overview.FailIps = topIpsByResult(since, models.LoginResultFail)
	overview.ActiveUsers = topUsers(since)
	overview.Suspicious = filterMinCount(overview.FailUsers, 5)
	return overview
}

// 分组统计的列名全部为字面量，取值走占位符，不接受外部输入拼接 SQL

func topUsersByResult(since string, result int8) []TopItem {
	var items []TopItem
	models.DB.Table("sys_login_log").
		Select("user_name as name, COUNT(*) as count").
		Where("created_at >= ? AND result = ?", since, result).
		Group("user_name").Order("count desc").Limit(10).Scan(&items)
	return nilToEmpty(items)
}

func topIpsByResult(since string, result int8) []TopItem {
	var items []TopItem
	models.DB.Table("sys_login_log").
		Select("ip_addr as name, COUNT(*) as count").
		Where("created_at >= ? AND result = ?", since, result).
		Group("ip_addr").Order("count desc").Limit(10).Scan(&items)
	return nilToEmpty(items)
}

func topUsers(since string) []TopItem {
	var items []TopItem
	models.DB.Table("sys_login_log").
		Select("user_name as name, COUNT(*) as count").
		Where("created_at >= ? AND result = ?", since, models.LoginResultSuccess).
		Group("user_name").Order("count desc").Limit(10).Scan(&items)
	return nilToEmpty(items)
}

func nilToEmpty(items []TopItem) []TopItem {
	if items == nil {
		return []TopItem{}
	}
	return items
}

func filterMinCount(items []TopItem, min int64) []TopItem {
	kept := make([]TopItem, 0, len(items))
	for _, item := range items {
		if item.Count >= min {
			kept = append(kept, item)
		}
	}
	return kept
}

// parseTimeInput 兼容 "2006-01-02 15:04:05" 与 "2006-01-02" 两种前端入参
func parseTimeInput(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 10 {
		return value + " 00:00:00"
	}
	return value
}

// parseBrowser 从 UA 粗粒度识别浏览器，仅用于审计展示
func parseBrowser(ua string) string {
	if ua == "" {
		return ""
	}
	rules := []struct{ keyword, name string }{
		{"Edg/", "Edge"},
		{"OPR/", "Opera"},
		{"MicroMessenger", "微信"},
		{"DingTalk", "钉钉"},
		{"Firefox", "Firefox"},
		{"Chrome", "Chrome"},
		{"Safari", "Safari"},
		{"curl", "curl"},
		{"python-requests", "python-requests"},
	}
	for _, rule := range rules {
		if strings.Contains(ua, rule.keyword) {
			return rule.name
		}
	}
	return "其他"
}

// parseOs 从 UA 粗粒度识别终端系统
func parseOs(ua string) string {
	if ua == "" {
		return ""
	}
	rules := []struct{ keyword, name string }{
		{"Windows NT", "Windows"},
		{"Mac OS X", "macOS"},
		{"Android", "Android"},
		{"iPhone", "iOS"},
		{"iPad", "iOS"},
		{"Linux", "Linux"},
	}
	for _, rule := range rules {
		if strings.Contains(ua, rule.keyword) {
			return rule.name
		}
	}
	return "其他"
}

func truncate(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
