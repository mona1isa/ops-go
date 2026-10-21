package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/monitor/mail"
	"github.com/zhany/ops-go/services/monitor/robot"
	"github.com/zhany/ops-go/services/monitor/webhook"
	"gorm.io/gorm"
)

// NotifierService 告警通知服务：按渠道投递并记录通知日志
type NotifierService struct{}

// enabledChannels 解析渠道ID列表（逗号分隔），返回启用中的渠道
func (n *NotifierService) enabledChannels(ids string) []models.OpsMonitorNotifyChannel {
	channels := make([]models.OpsMonitorNotifyChannel, 0)
	for _, field := range strings.Split(ids, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil || id <= 0 {
			continue
		}
		var channel models.OpsMonitorNotifyChannel
		if err := models.DB.Where("id = ? AND del_flag = ? AND enabled = ?", id, "0", "1").First(&channel).Error; err != nil {
			log.Printf("告警通知渠道不可用 id=%d: %v", id, err)
			continue
		}
		channels = append(channels, channel)
	}
	return channels
}

// Dispatch 向规则配置的渠道投递告警或恢复通知。
// recovered 为 true 表示恢复通知，可绕过渠道未启用之外的所有限制。
func (n *NotifierService) Dispatch(ctx context.Context, event *models.OpsMonitorAlertEvent, channelIds string, recovered bool) int {
	channels := n.enabledChannels(channelIds)
	if len(channels) == 0 {
		return 0
	}
	payload := n.buildPayload(event, recovered)

	success := 0
	for i := range channels {
		channel := channels[i]
		if err := n.sendToChannel(ctx, event, &channel, payload, recovered); err != nil {
			log.Printf("告警通知投递失败 event=%d channel=%d: %v", event.ID, channel.ID, err)
			continue
		}
		success++
	}
	return success
}

// sendToChannel 单个渠道投递，失败由各自 Dispatch 内部重试，每次尝试落一条通知日志
func (n *NotifierService) sendToChannel(ctx context.Context, event *models.OpsMonitorAlertEvent, channel *models.OpsMonitorNotifyChannel, payload *webhook.Payload, recovered bool) error {
	onAttempt := func(item webhook.Attempt) {
		n.saveLog(event, channel, item, recovered, payload)
	}
	switch channel.Type {
	case models.ChannelTypeWebhook:
		body, err := webhook.BuildBody(payload)
		if err != nil {
			return err
		}
		request := &webhook.Request{
			Url:     channel.Url,
			Secret:  channel.Secret,
			Headers: parseHeaders(channel.Headers),
			Body:    body,
			EventId: event.ID,
		}
		attempt, err := webhook.Dispatch(ctx, request, onAttempt)
		if err != nil {
			return err
		}
		if !attempt.Success {
			return errors.New("通知渠道未成功接收")
		}
		return nil
	case models.ChannelTypeDingTalk, models.ChannelTypeWeCom, models.ChannelTypeFeishu:
		attempt, err := robot.Dispatch(ctx, robot.Kind(channel.Type), payload, channel.Url, channel.Secret, webhook.MaxAttempts, onAttempt)
		if err != nil {
			return err
		}
		if !attempt.Success {
			return errors.New(attempt.ErrorMsg)
		}
		return nil
	case models.ChannelTypeEmail:
		cfg, err := mail.ParseConfig(channel.Headers)
		if err != nil {
			return err
		}
		recipients, err := mail.SplitRecipients(channel.Url)
		if err != nil {
			return err
		}
		attempt, err := mail.Send(ctx, cfg, recipients, payload, webhook.MaxAttempts, onAttempt)
		if err != nil {
			return err
		}
		if !attempt.Success {
			return errors.New(attempt.ErrorMsg)
		}
		return nil
	default:
		return fmt.Errorf("未支持的通知渠道类型: %d", channel.Type)
	}
}

// TestSend 渠道试发送，不落告警事件，仅返回投递结果描述
func (n *NotifierService) TestSend(ctx context.Context, channel *models.OpsMonitorNotifyChannel) (bool, string, error) {
	payload := &webhook.Payload{
		EventId:      0,
		RuleName:     "通知渠道连通性测试",
		Severity:     models.SeverityInfo,
		SeverityText: webhook.SeverityText(models.SeverityInfo),
		Status:       "testing",
		Summary:      "这是一条来自运维平台的测试通知",
		Instance:     "-",
		Metric:       "-",
		FiredAt:      time.Now().Format("2006-01-02 15:04:05"),
		Labels:       map[string]string{},
	}
	switch channel.Type {
	case models.ChannelTypeWebhook:
		body, err := webhook.BuildBody(payload)
		if err != nil {
			return false, "", err
		}
		request := &webhook.Request{
			Url:     channel.Url,
			Secret:  channel.Secret,
			Headers: parseHeaders(channel.Headers),
			Body:    body,
		}
		attempt, err := webhook.Dispatch(ctx, request, nil)
		if err != nil {
			return false, "", err
		}
		return attempt.Success, fmt.Sprintf("HTTP %d %s", attempt.StatusCode, truncate(attempt.Body, 200)), nil
	case models.ChannelTypeDingTalk, models.ChannelTypeWeCom, models.ChannelTypeFeishu:
		attempt, err := robot.Dispatch(ctx, robot.Kind(channel.Type), payload, channel.Url, channel.Secret, 1, nil)
		if err != nil {
			return false, "", err
		}
		message := truncate(attempt.Body, 200)
		if attempt.ErrorMsg != "" {
			message = attempt.ErrorMsg
		}
		return attempt.Success, message, nil
	case models.ChannelTypeEmail:
		cfg, err := mail.ParseConfig(channel.Headers)
		if err != nil {
			return false, "", err
		}
		recipients, err := mail.SplitRecipients(channel.Url)
		if err != nil {
			return false, "", err
		}
		attempt, err := mail.Send(ctx, cfg, recipients, payload, 1, nil)
		if err != nil {
			return false, "", err
		}
		message := attempt.Body
		if attempt.ErrorMsg != "" {
			message = attempt.ErrorMsg
		}
		return attempt.Success, message, nil
	default:
		return false, "", fmt.Errorf("未支持的通知渠道类型")
	}
}

func (n *NotifierService) buildPayload(event *models.OpsMonitorAlertEvent, recovered bool) *webhook.Payload {
	firedAt := ""
	if event.FiredAt != nil {
		firedAt = event.FiredAt.Format("2006-01-02 15:04:05")
	}
	status := "firing"
	if recovered {
		status = "resolved"
	}
	labels := parseLabels(event.Labels)
	return &webhook.Payload{
		EventId:      event.ID,
		RuleId:       event.RuleId,
		RuleName:     event.RuleName,
		Severity:     event.Severity,
		SeverityText: webhook.SeverityText(event.Severity),
		Status:       status,
		Recovered:    recovered,
		Instance:     event.InstanceIp,
		Metric:       labels["__name__"],
		Value:        event.Value,
		Threshold:    event.Threshold,
		Summary:      event.Summary,
		FiredAt:      firedAt,
		Labels:       labels,
	}
}

func (n *NotifierService) saveLog(event *models.OpsMonitorAlertEvent, channel *models.OpsMonitorNotifyChannel, attempt webhook.Attempt, recovered bool, payload *webhook.Payload) {
	body, _ := webhook.BuildBody(payload)
	record := &models.OpsMonitorNotifyLog{
		EventId:      event.ID,
		RuleId:       event.RuleId,
		ChannelId:    channel.ID,
		ChannelName:  channel.Name,
		Recovered:    boolToInt8(recovered),
		Payload:      string(body),
		StatusCode:   attempt.StatusCode,
		ResponseBody: truncate(attempt.Body, 1000),
		Attempt:      attempt.Attempt,
		Success:      boolToInt8(attempt.Success),
		ErrorMsg:     truncate(attempt.ErrorMsg, 500),
	}
	if err := models.DB.Create(record).Error; err != nil {
		log.Printf("写入告警通知日志失败 event=%d: %v", event.ID, err)
	}
}

func boolToInt8(value bool) int8 {
	if value {
		return 1
	}
	return 0
}

// parseLabels 解析事件标签集快照，失败返回空集合（不影响通知投递）
func parseLabels(raw string) map[string]string {
	labels := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return labels
	}
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return map[string]string{}
	}
	return labels
}

// parseHeaders 解析渠道自定义请求头（JSON 对象），失败返回空集合
func parseHeaders(raw string) map[string]string {
	headers := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return headers
	}
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		return map[string]string{}
	}
	return headers
}

// recentNotifyLog 供事件列表展示最近一次通知结果
func recentNotifyLog(eventId uint64) *models.OpsMonitorNotifyLog {
	var record models.OpsMonitorNotifyLog
	if err := models.DB.Where("event_id = ?", eventId).Order("id desc").First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		log.Println("查询告警通知日志异常:", err)
		return nil
	}
	return &record
}
