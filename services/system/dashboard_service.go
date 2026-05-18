package system

import (
	"fmt"
	"github.com/zhany/ops-go/bastion"
	"github.com/zhany/ops-go/models"
	"time"
)

type DashboardService struct{}

// DashboardStats 首页仪表盘统计数据
type DashboardStats struct {
	// 顶部卡片统计
	InstanceCount       int64   `json:"instanceCount"`       // 服务器实例总数
	ActiveSessionCount  int     `json:"activeSessionCount"`  // 在线会话数
	MonthSessionCount   int64   `json:"monthSessionCount"`   // 本月会话数
	DangerousRuleCount  int64   `json:"dangerousRuleCount"`  // 危险命令规则数

	// 快捷导航统计
	UserCount           int64   `json:"userCount"`           // 用户总数
	GroupCount          int64   `json:"groupCount"`          // 主机分组数
	KeyCount            int64   `json:"keyCount"`            // 密钥数
	ScriptCount         int64   `json:"scriptCount"`         // 脚本数
	TaskTemplateCount   int64   `json:"taskTemplateCount"`   // 任务模板数
	TotalSessionCount   int64   `json:"totalSessionCount"`   // 总会话数
	OnlineInstanceCount int64   `json:"onlineInstanceCount"` // 在线主机数

	// 趋势数据
	SessionTrend        []TrendItem `json:"sessionTrend"`      // 最近30天会话趋势
	GroupDistribution   []GroupDistItem `json:"groupDistribution"` // 主机分组分布
}

// TrendItem 趋势数据项
type TrendItem struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// GroupDistItem 分组分布数据项
type GroupDistItem struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// GetStats 获取首页仪表盘统计数据
func (s *DashboardService) GetStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	// 1. 服务器实例总数
	if err := models.DB.Model(&models.OpsInstance{}).Count(&stats.InstanceCount).Error; err != nil {
		return nil, fmt.Errorf("查询服务器实例数失败: %w", err)
	}

	// 2. 在线会话数（从内存中的 SessionManager 获取，更准确）
	sm := bastion.GetSessionManager()
	stats.ActiveSessionCount = len(sm.ListSessions())

	// 3. 本月会话数
	now := time.Now()
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := models.DB.Model(&models.OpsSessionRecord{}).
		Where("start_time >= ?", firstDayOfMonth).Count(&stats.MonthSessionCount).Error; err != nil {
		return nil, fmt.Errorf("查询本月会话数失败: %w", err)
	}

	// 4. 危险命令规则数（启用的）
	if err := models.DB.Model(&models.OpsDangerousCommand{}).
		Where("is_enabled = ?", 1).Count(&stats.DangerousRuleCount).Error; err != nil {
		return nil, fmt.Errorf("查询危险命令规则数失败: %w", err)
	}

	// 5. 用户总数
	if err := models.DB.Model(&models.SysUser{}).Count(&stats.UserCount).Error; err != nil {
		return nil, fmt.Errorf("查询用户总数失败: %w", err)
	}

	// 6. 主机分组数
	if err := models.DB.Model(&models.OpsGroup{}).Count(&stats.GroupCount).Error; err != nil {
		return nil, fmt.Errorf("查询主机分组数失败: %w", err)
	}

	// 7. 密钥数
	if err := models.DB.Model(&models.OpsKey{}).Count(&stats.KeyCount).Error; err != nil {
		return nil, fmt.Errorf("查询密钥数失败: %w", err)
	}

	// 8. 脚本数
	if err := models.DB.Model(&models.OpsScript{}).Count(&stats.ScriptCount).Error; err != nil {
		return nil, fmt.Errorf("查询脚本数失败: %w", err)
	}

	// 9. 任务模板数
	if err := models.DB.Model(&models.OpsTaskTemplate{}).Count(&stats.TaskTemplateCount).Error; err != nil {
		return nil, fmt.Errorf("查询任务模板数失败: %w", err)
	}

	// 10. 总会话数
	if err := models.DB.Model(&models.OpsSessionRecord{}).Count(&stats.TotalSessionCount).Error; err != nil {
		return nil, fmt.Errorf("查询总会话数失败: %w", err)
	}

	// 11. 在线主机数
	if err := models.DB.Model(&models.OpsInstance{}).
		Where("online_status = ?", "1").Count(&stats.OnlineInstanceCount).Error; err != nil {
		return nil, fmt.Errorf("查询在线主机数失败: %w", err)
	}

	// 12. 最近30天会话趋势
	trend, err := s.getSessionTrend(30)
	if err != nil {
		return nil, err
	}
	stats.SessionTrend = trend

	// 13. 主机分组分布
	groupDist, err := s.getGroupDistribution()
	if err != nil {
		return nil, err
	}
	stats.GroupDistribution = groupDist

	return stats, nil
}

// getSessionTrend 获取最近 N 天的会话趋势
func (s *DashboardService) getSessionTrend(days int) ([]TrendItem, error) {
	result := make([]TrendItem, days)
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -(days - 1 - i))
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.AddDate(0, 0, 1)

		var count int64
		if err := models.DB.Model(&models.OpsSessionRecord{}).
			Where("start_time >= ? AND start_time < ?", startOfDay, endOfDay).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("查询会话趋势失败: %w", err)
		}

		result[i] = TrendItem{
			Date:  date.Format("01-02"),
			Count: count,
		}
	}

	return result, nil
}

// getGroupDistribution 获取主机分组分布
func (s *DashboardService) getGroupDistribution() ([]GroupDistItem, error) {
	type groupCount struct {
		GroupId int   `gorm:"column:group_id"`
		Count   int64 `gorm:"column:count"`
	}

	var results []groupCount
	// 查询 ops_instance_group 关联表中的分组统计
	if err := models.DB.Model(&models.OpsInstanceGroup{}).
		Select("group_id, COUNT(*) as count").
		Group("group_id").
		Scan(&results).Error; err != nil {
		// 如果表不存在或没有数据，尝试用 ops_user_instance_auth 中的 group_id
		return s.getGroupDistributionFromAuth(), nil
	}

	if len(results) == 0 {
		return s.getGroupDistributionFromAuth(), nil
	}

	// 获取分组名称
	groupMap := make(map[int]string)
	var groups []models.OpsGroup
	if err := models.DB.Find(&groups).Error; err == nil {
		for _, g := range groups {
			groupMap[g.ID] = g.Name
		}
	}

	dist := make([]GroupDistItem, 0, len(results))
	for _, r := range results {
		name := groupMap[r.GroupId]
		if name == "" {
			name = fmt.Sprintf("分组%d", r.GroupId)
		}
		dist = append(dist, GroupDistItem{
			Name:  name,
			Value: r.Count,
		})
	}

	return dist, nil
}

// getGroupDistributionFromAuth 从授权表获取分组分布（兜底方案）
func (s *DashboardService) getGroupDistributionFromAuth() []GroupDistItem {
	type groupAuthCount struct {
		GroupId int   `gorm:"column:group_id"`
		Count   int64 `gorm:"column:count"`
	}

	var results []groupAuthCount
	if err := models.DB.Model(&models.OpsUserInstanceAuth{}).
		Where("group_id > 0 AND auth_type = 2").
		Select("group_id, COUNT(DISTINCT user_id) as count").
		Group("group_id").
		Scan(&results).Error; err != nil {
		return []GroupDistItem{}
	}

	groupMap := make(map[int]string)
	var groups []models.OpsGroup
	if err := models.DB.Find(&groups).Error; err == nil {
		for _, g := range groups {
			groupMap[g.ID] = g.Name
		}
	}

	dist := make([]GroupDistItem, 0, len(results))
	for _, r := range results {
		name := groupMap[r.GroupId]
		if name == "" {
			name = fmt.Sprintf("分组%d", r.GroupId)
		}
		dist = append(dist, GroupDistItem{
			Name:  name,
			Value: r.Count,
		})
	}

	return dist
}
