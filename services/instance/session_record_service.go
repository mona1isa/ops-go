package instance

import (
	"bufio"
	"errors"
	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
	"log"
	"os"
	"strings"
	"time"
)

type SessionRecordService struct{}

// ListRequest 查询会话记录列表请求
type ListRequest struct {
	Page         int    `form:"page" json:"page"`
	PageSize     int    `form:"pageSize" json:"pageSize"`
	UserId       int    `form:"userId" json:"userId"`
	InstanceId   int    `form:"instanceId" json:"instanceId"`
	InstanceName string `form:"instanceName" json:"instanceName"`
	InstanceIp   string `form:"instanceIp" json:"instanceIp"`
	KeyUser      string `form:"keyUser" json:"keyUser"`
	Status       *int8  `form:"status" json:"status"`
	StartTime    string `form:"startTime" json:"startTime"`
	EndTime      string `form:"endTime" json:"endTime"`
}

// ListResponse 会话记录列表响应
type ListResponse struct {
	Total int                        `json:"total"`
	List  []models.OpsSessionRecord  `json:"list"`
}

// List 查询会话记录列表
func (s *SessionRecordService) List(req *ListRequest) (*ListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	db := models.DB.Model(&models.OpsSessionRecord{})

	// 条件过滤
	if req.UserId > 0 {
		db = db.Where("user_id = ?", req.UserId)
	}
	if req.InstanceId > 0 {
		db = db.Where("instance_id = ?", req.InstanceId)
	}
	if req.InstanceName != "" {
		db = db.Where("instance_name LIKE ?", "%"+req.InstanceName+"%")
	}
	if req.InstanceIp != "" {
		db = db.Where("instance_ip LIKE ?", "%"+req.InstanceIp+"%")
	}
	if req.KeyUser != "" {
		db = db.Where("key_user LIKE ?", "%"+req.KeyUser+"%")
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if req.StartTime != "" {
		db = db.Where("start_time >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		db = db.Where("start_time <= ?", req.EndTime)
	}

	// 查询总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 查询列表
	var records []models.OpsSessionRecord
	offset := (req.Page - 1) * req.PageSize
	if err := db.Order("start_time DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&records).Error; err != nil {
		return nil, err
	}

	return &ListResponse{
		Total: int(total),
		List:  records,
	}, nil
}

// GetByID 根据ID查询会话记录
func (s *SessionRecordService) GetByID(id uint64) (*models.OpsSessionRecord, error) {
	var record models.OpsSessionRecord
	if err := models.DB.First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话记录不存在")
		}
		return nil, err
	}
	return &record, nil
}

// GetBySessionID 根据会话ID查询会话记录
func (s *SessionRecordService) GetBySessionID(sessionID string) (*models.OpsSessionRecord, error) {
	var record models.OpsSessionRecord
	if err := models.DB.Where("session_id = ?", sessionID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话记录不存在")
		}
		return nil, err
	}
	return &record, nil
}

// Delete 删除会话记录（同时删除录像文件）
func (s *SessionRecordService) Delete(id uint64) error {
	_, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 删除数据库记录
	if err := models.DB.Delete(&models.OpsSessionRecord{}, id).Error; err != nil {
		return err
	}

	// TODO: 删除录像文件（如果需要）
	// if record.RecordingFile != "" {
	//     os.Remove(record.RecordingFile)
	// }

	return nil
}

// Statistics 统计数据
type Statistics struct {
	TotalSessions   int64 `json:"totalSessions"`
	ActiveSessions  int64 `json:"activeSessions"`
	TotalDuration   int64 `json:"totalDuration"`
	AverageDuration int64 `json:"averageDuration"`
}

// GetStatistics 获取统计数据
func (s *SessionRecordService) GetStatistics(userId int) (*Statistics, error) {
	stats := &Statistics{}

	db := models.DB.Model(&models.OpsSessionRecord{})
	if userId > 0 {
		db = db.Where("user_id = ?", userId)
	}

	// 总会话数
	db.Count(&stats.TotalSessions)

	// 活跃会话数
	models.DB.Model(&models.OpsSessionRecord{}).
		Where("status = ?", models.SessionStatusActive).
		Count(&stats.ActiveSessions)

	// 总时长
	var totalDuration int64
	models.DB.Model(&models.OpsSessionRecord{}).
		Where("status = ?", models.SessionStatusCompleted).
		Select("COALESCE(SUM(duration), 0)").
		Scan(&totalDuration)
	stats.TotalDuration = totalDuration

	// 平均时长
	if stats.TotalSessions > 0 {
		stats.AverageDuration = totalDuration / stats.TotalSessions
	}

	return stats, nil
}

// GetPlaybackContent 读取录像文件内容
func (s *SessionRecordService) GetPlaybackContent(filePath string) (string, error) {
	// 使用逐行读取的方式，确保每行 JSON 数据完整
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 使用 Scanner 逐行读取
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	// 返回所有行，用 \n 连接
	return strings.Join(lines, "\n"), nil
}

// ListActiveSessions 列出活跃会话
func (s *SessionRecordService) ListActiveSessions() ([]map[string]interface{}, error) {
	// 从数据库查询活跃会话
	var records []models.OpsSessionRecord
	if err := models.DB.Where("status = ?", models.SessionStatusActive).
		Order("start_time DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	sessions := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, map[string]interface{}{
			"id":            record.ID,
			"sessionId":     record.SessionID,
			"userId":        record.UserID,
			"instanceId":    record.InstanceID,
			"instanceName":  record.InstanceName,
			"instanceIp":    record.InstanceIP,
			"keyUser":       record.KeyUser,
			"startTime":     record.StartTime,
			"duration":      int(time.Since(record.StartTime).Seconds()),
			"status":        record.Status,
		})
	}

	return sessions, nil
}

// SessionTerminator 接口，用于终止会话连接
type SessionTerminator interface {
	Terminate(sessionID string) error
}

// 终止器列表（支持多个终止器）
var terminators []SessionTerminator

// RegisterTerminator 注册会话终止器
func RegisterTerminator(terminator SessionTerminator) {
	terminators = append(terminators, terminator)
}

// TerminateSession 终止会话
func (s *SessionRecordService) TerminateSession(sessionID string, isAdmin bool, currentUserId int) error {
	// 查询会话记录
	var record models.OpsSessionRecord
	if err := models.DB.Where("session_id = ?", sessionID).First(&record).Error; err != nil {
		return errors.New("会话不存在")
	}

	// 检查会话状态
	if record.Status != models.SessionStatusActive {
		return errors.New("会话已结束")
	}

	// 尝试通过所有终止器关闭连接
	for _, terminator := range terminators {
		if err := terminator.Terminate(sessionID); err != nil {
			log.Printf("通过终止器关闭会话失败: %v", err)
		} else {
			// 至少有一个终止器成功终止会话
			// 某些终止器（如 Bastion）内部已经更新了数据库，但为了保险，下面还是会更新
			break
		}
	}

	// 更新数据库状态
	now := time.Now()
	duration := int(now.Sub(record.StartTime).Seconds())

	if err := models.DB.Model(&models.OpsSessionRecord{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"end_time": &now,
			"duration":  duration,
			"status":    models.SessionStatusAborted,
		}).Error; err != nil {
		return err
	}

	return nil
}
