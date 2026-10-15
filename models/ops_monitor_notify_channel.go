package models

// 通知渠道类型（2/3/4 为后续邮件、钉钉、企业微信预留）
const (
	ChannelTypeWebhook  = 1 // 自定义 Webhook
	ChannelTypeEmail    = 2 // 邮件
	ChannelTypeDingTalk = 3 // 钉钉
	ChannelTypeWeCom    = 4 // 企业微信
)

// OpsMonitorNotifyChannel 告警通知渠道
type OpsMonitorNotifyChannel struct {
	Name    string `gorm:"column:name;type:varchar(128);not null;comment:渠道名称" json:"name"`
	Type    int8   `gorm:"column:type;type:tinyint;default:1;comment:类型（1 Webhook 2 邮件 3 钉钉 4 企业微信）" json:"type"`
	Url     string `gorm:"column:url;type:varchar(512);not null;comment:回调地址" json:"url"`
	Secret  string `gorm:"column:secret;type:varchar(255);comment:HMAC-SHA256 签名密钥" json:"secret"`
	Headers string `gorm:"column:headers;type:text;comment:自定义请求头（JSON 对象）" json:"headers"`
	Enabled string `gorm:"column:enabled;type:varchar(1);default:1;comment:启用状态（1 启用 0 停用）" json:"enabled"`
	Base
}

const TableOpsMonitorNotifyChannel = "ops_monitor_notify_channel"

func (OpsMonitorNotifyChannel) TableName() string {
	return TableOpsMonitorNotifyChannel
}
