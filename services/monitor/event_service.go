package monitor

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// EventService 告警事件查询与人工处理服务
type EventService struct{}

// EventFilter 事件列表查询条件
type EventFilter struct {
	RuleId     *int
	Severity   *int8
	Status     *int8
	InstanceIp string
	Source     string
	StartAt    string
	EndAt      string
}

// EventView 事件列表项，附带最近一次通知结果
type EventView struct {
	models.OpsMonitorAlertEvent
	LastNotify *models.OpsMonitorNotifyLog `json:"lastNotify"`
}

// List 分页查询告警事件
func (s *EventService) List(pageNum, pageSize int, filter EventFilter) (models.PageResult[EventView], error) {
	result, err := models.Paginate[models.OpsMonitorAlertEvent](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		if filter.RuleId != nil {
			db = db.Where("rule_id = ?", *filter.RuleId)
		}
		if filter.Severity != nil {
			db = db.Where("severity = ?", *filter.Severity)
		}
		if filter.Status != nil {
			db = db.Where("status = ?", *filter.Status)
		}
		if filter.InstanceIp != "" {
			db = db.Where("instance_ip like ?", "%"+escapeLike(filter.InstanceIp)+"%")
		}
		// 默认只看指标告警，避免日志告警混入监控事件页；all 表示不分来源
		switch filter.Source {
		case "", models.AlertSourceMonitor:
			db = db.Where("source IS NULL OR source = '' OR source = ?", models.AlertSourceMonitor)
		case "all":
		default:
			db = db.Where("source = ?", filter.Source)
		}
		if filter.StartAt != "" {
			db = db.Where("created_at >= ?", filter.StartAt)
		}
		if filter.EndAt != "" {
			db = db.Where("created_at <= ?", filter.EndAt)
		}
		return db.Order("id desc")
	})
	if err != nil {
		log.Println("查询告警事件失败：", err)
		return models.PageResult[EventView]{}, errors.New("查询告警事件失败")
	}

	page := models.PageResult[EventView]{
		Total:     result.Total,
		PageNum:   result.PageNum,
		PageSize:  result.PageSize,
		TotalPage: result.TotalPage,
		Data:      make([]EventView, 0, len(result.Data)),
	}
	ids := make([]uint64, 0, len(result.Data))
	for i := range result.Data {
		ids = append(ids, result.Data[i].ID)
	}
	latest := latestNotifyLogs(ids)
	for i := range result.Data {
		page.Data = append(page.Data, EventView{
			OpsMonitorAlertEvent: result.Data[i],
			LastNotify:           latest[result.Data[i].ID],
		})
	}
	return page, nil
}

// Detail 事件详情及其通知历史
func (s *EventService) Detail(id uint64) (*models.OpsMonitorAlertEvent, []models.OpsMonitorNotifyLog, error) {
	var event models.OpsMonitorAlertEvent
	if err := models.DB.First(&event, id).Error; err != nil {
		return nil, nil, errors.New("告警事件不存在")
	}
	logs := make([]models.OpsMonitorNotifyLog, 0)
	if err := models.DB.Where("event_id = ?", id).Order("id desc").Limit(50).Find(&logs).Error; err != nil {
		log.Println("查询告警通知记录失败：", err)
	}
	for i := range logs {
		logs[i].Payload = ""
	}
	return &event, logs, nil
}

// Ack 人工处理告警事件：确认或关闭
func (s *EventService) Ack(id uint64, action string, remark string, operator string) error {
	var event models.OpsMonitorAlertEvent
	if err := models.DB.First(&event, id).Error; err != nil {
		return errors.New("告警事件不存在")
	}
	if event.Status == models.AlertStatusResolved {
		return errors.New("该告警已恢复，无需处理")
	}

	now := time.Now()
	switch strings.TrimSpace(action) {
	case "confirm":
		if err := models.DB.Model(&event).Updates(map[string]interface{}{
			"suppressed": event.Suppressed,
			"summary":    appendRemark(event.Summary, remark, operator),
			"updated_at": now,
		}).Error; err != nil {
			log.Println("确认告警事件失败：", err)
			return errors.New("确认告警失败")
		}
		return nil
	case "close":
		if err := models.DB.Model(&event).Updates(map[string]interface{}{
			"status":      models.AlertStatusResolved,
			"resolved_at": &now,
			"summary":     appendRemark(event.Summary, remark, operator),
			"updated_at":  now,
		}).Error; err != nil {
			log.Println("关闭告警事件失败：", err)
			return errors.New("关闭告警失败")
		}
		return nil
	default:
		return errors.New("处理动作仅支持 confirm 或 close")
	}
}

// Delete 删除事件记录
func (s *EventService) Delete(id uint64) error {
	var event models.OpsMonitorAlertEvent
	if err := models.DB.First(&event, id).Error; err != nil {
		return errors.New("告警事件不存在")
	}
	if event.Status != models.AlertStatusResolved {
		return errors.New("仅已恢复的告警可以删除")
	}
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("event_id = ?", id).Delete(&models.OpsMonitorNotifyLog{}).Error; err != nil {
			return err
		}
		return tx.Delete(&event).Error
	})
	if err != nil {
		log.Println("删除告警事件失败：", err)
		return errors.New("删除告警事件失败")
	}
	return nil
}

// latestNotifyLogs 批量取每个事件最近一次通知记录
func latestNotifyLogs(ids []uint64) map[uint64]*models.OpsMonitorNotifyLog {
	result := map[uint64]*models.OpsMonitorNotifyLog{}
	if len(ids) == 0 {
		return result
	}
	var records []models.OpsMonitorNotifyLog
	if err := models.DB.Where("event_id IN ?", ids).Order("id desc").Find(&records).Error; err != nil {
		log.Println("查询告警通知记录失败：", err)
		return result
	}
	for i := range records {
		record := records[i]
		// 通知内容可能含业务信息，列表只回传投递结果
		record.Payload = ""
		if _, exists := result[record.EventId]; exists {
			continue
		}
		result[record.EventId] = &record
	}
	return result
}

func appendRemark(summary string, remark string, operator string) string {
	remark = strings.TrimSpace(remark)
	if remark == "" {
		return summary
	}
	text := fmt.Sprintf("%s ｜ 处理：%s", summary, remark)
	if operator != "" {
		text = fmt.Sprintf("%s（by %s）", text, operator)
	}
	return truncate(text, 500)
}
