package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/monitor/mail"
	"github.com/zhany/ops-go/services/monitor/robot"
	"gorm.io/gorm"
)

// SecretMask 渠道密钥在接口返回值中的占位符，永不回传真实密钥
const SecretMask = "******"

// ChannelService 告警通知渠道管理服务
type ChannelService struct{}

// ChannelFilter 渠道列表查询条件
type ChannelFilter struct {
	Name    string
	Type    *int8
	Enabled string
}

// List 分页查询通知渠道（密钥脱敏）
func (s *ChannelService) List(pageNum, pageSize int, filter ChannelFilter) (models.PageResult[models.OpsMonitorNotifyChannel], error) {
	result, err := models.Paginate[models.OpsMonitorNotifyChannel](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		db = db.Where("del_flag = ?", 0)
		if filter.Name != "" {
			db = db.Where("name like ?", "%"+escapeLike(filter.Name)+"%")
		}
		if filter.Type != nil {
			db = db.Where("type = ?", *filter.Type)
		}
		if filter.Enabled != "" {
			db = db.Where("enabled = ?", filter.Enabled)
		}
		return db.Order("id desc")
	})
	if err != nil {
		log.Println("查询通知渠道失败：", err)
		return result, errors.New("查询通知渠道失败")
	}
	for i := range result.Data {
		maskSecret(&result.Data[i])
	}
	return result, nil
}

// Options 启用中的渠道下拉列表（密钥脱敏）
func (s *ChannelService) Options() ([]models.OpsMonitorNotifyChannel, error) {
	var channels []models.OpsMonitorNotifyChannel
	if err := models.DB.Where("del_flag = ? AND enabled = ?", 0, "1").Order("id asc").Find(&channels).Error; err != nil {
		log.Println("查询通知渠道失败：", err)
		return nil, errors.New("查询通知渠道失败")
	}
	for i := range channels {
		maskSecret(&channels[i])
	}
	return channels, nil
}

// Add 新增通知渠道
func (s *ChannelService) Add(channel *models.OpsMonitorNotifyChannel) error {
	if err := validateChannel(channel); err != nil {
		return err
	}
	if err := models.DB.Save(channel).Error; err != nil {
		log.Println("新增通知渠道失败：", err)
		return errors.New("新增通知渠道失败")
	}
	maskSecret(channel)
	return nil
}

// Edit 编辑通知渠道，密钥传空表示保持原值
func (s *ChannelService) Edit(channel *models.OpsMonitorNotifyChannel) error {
	var existing models.OpsMonitorNotifyChannel
	if err := models.DB.Where("id = ? AND del_flag = ?", channel.ID, 0).First(&existing).Error; err != nil {
		return errors.New("通知渠道不存在")
	}
	if strings.TrimSpace(channel.Secret) == "" || channel.Secret == SecretMask {
		channel.Secret = existing.Secret
	}
	if channel.Type == models.ChannelTypeEmail {
		channel.Headers = mail.RestorePassword(channel.Headers, existing.Headers, SecretMask)
	}
	if err := validateChannel(channel); err != nil {
		return err
	}
	channel.CreateBy = existing.CreateBy
	updates := map[string]interface{}{
		"name":       channel.Name,
		"type":       channel.Type,
		"url":        channel.Url,
		"secret":     channel.Secret,
		"headers":    channel.Headers,
		"enabled":    channel.Enabled,
		"remark":     channel.Remark,
		"update_by":  channel.UpdateBy,
		"updated_at": time.Now(),
	}
	if err := models.DB.Model(&existing).Updates(updates).Error; err != nil {
		log.Println("编辑通知渠道失败：", err)
		return errors.New("编辑通知渠道失败")
	}
	maskSecret(channel)
	return nil
}

// Delete 软删除通知渠道
func (s *ChannelService) Delete(id int) error {
	var existing models.OpsMonitorNotifyChannel
	if err := models.DB.Where("id = ? AND del_flag = ?", id, 0).First(&existing).Error; err != nil {
		return errors.New("通知渠道不存在")
	}
	if err := models.DB.Model(&existing).Update("del_flag", 1).Error; err != nil {
		log.Println("删除通知渠道失败：", err)
		return errors.New("删除通知渠道失败")
	}
	return nil
}

// Toggle 启用/停用通知渠道
func (s *ChannelService) Toggle(id int, enabled string) error {
	if enabled != "0" && enabled != "1" {
		return errors.New("启用状态非法")
	}
	if err := models.DB.Model(&models.OpsMonitorNotifyChannel{}).Where("id = ? AND del_flag = ?", id, 0).
		Update("enabled", enabled).Error; err != nil {
		log.Println("更新通知渠道状态失败：", err)
		return errors.New("更新通知渠道状态失败")
	}
	return nil
}

// ChannelTestResult 渠道测试发送结果
type ChannelTestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Test 用真实渠道配置发送一条测试通知
func (s *ChannelService) Test(ctx context.Context, id int) (*ChannelTestResult, error) {
	var channel models.OpsMonitorNotifyChannel
	if err := models.DB.Where("id = ? AND del_flag = ?", id, 0).First(&channel).Error; err != nil {
		return nil, errors.New("通知渠道不存在")
	}
	ok, message, err := (&NotifierService{}).TestSend(ctx, &channel)
	if err != nil {
		return nil, err
	}
	result := &ChannelTestResult{Success: ok, Message: message}
	if !ok {
		result.Message = "发送失败：" + message
	}
	return result, nil
}

func validateChannel(channel *models.OpsMonitorNotifyChannel) error {
	channel.Name = strings.TrimSpace(channel.Name)
	channel.Url = strings.TrimSpace(channel.Url)
	if channel.Name == "" {
		return errors.New("渠道名称不能为空")
	}
	if len([]rune(channel.Name)) > 128 {
		return errors.New("渠道名称过长")
	}
	switch channel.Type {
	case models.ChannelTypeWebhook:
		if channel.Url == "" {
			return errors.New("通知地址不能为空")
		}
		if !strings.HasPrefix(channel.Url, "http://") && !strings.HasPrefix(channel.Url, "https://") {
			return errors.New("通知地址仅支持 http/https")
		}
		if len(channel.Url) > 512 {
			return errors.New("通知地址过长")
		}
	case models.ChannelTypeDingTalk, models.ChannelTypeWeCom, models.ChannelTypeFeishu:
		if err := robot.ValidateUrl(robot.Kind(channel.Type), channel.Url); err != nil {
			return err
		}
		if channel.Type == models.ChannelTypeWeCom {
			channel.Secret = ""
		}
	case models.ChannelTypeEmail:
		if _, err := mail.ParseConfig(channel.Headers); err != nil {
			return err
		}
		if _, err := mail.SplitRecipients(channel.Url); err != nil {
			return err
		}
		if len(channel.Url) > 512 {
			return errors.New("收件人列表过长")
		}
		channel.Secret = ""
	default:
		return errors.New("通知渠道类型不合法")
	}
	if channel.Type == models.ChannelTypeWebhook {
		if headers := strings.TrimSpace(channel.Headers); headers != "" {
			parsed := map[string]string{}
			if err := json.Unmarshal([]byte(headers), &parsed); err != nil {
				return errors.New("自定义请求头需为 JSON 对象，如 {\"X-Team\":\"sre\"}")
			}
			channel.Headers = headers
		}
	}
	if len(channel.Secret) > 255 {
		return errors.New("签名密钥过长")
	}
	if channel.Enabled != "0" && channel.Enabled != "1" {
		channel.Enabled = "1"
	}
	return nil
}

func maskSecret(channel *models.OpsMonitorNotifyChannel) {
	if channel.Secret != "" {
		channel.Secret = SecretMask
	} else {
		channel.Secret = ""
	}
	if channel.Type == models.ChannelTypeEmail {
		channel.Headers = mail.MaskPassword(channel.Headers, SecretMask)
	}
}
