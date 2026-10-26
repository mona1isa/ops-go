package models

// OpsCertAlertRule 证书/域名到期告警规则：扫描资产台账中带到期时间的记录，
// 剩余天数进入预警窗口即触发，复用监控告警事件表（source=cert）与通知链路。
type OpsCertAlertRule struct {
	Name         string `gorm:"column:name;type:varchar(128);not null;comment:规则名称" json:"name"`
	ModelTypes   string `gorm:"column:model_types;type:varchar(128);comment:监控的资产模型类型(逗号分隔，如 ssl,domain；空为所有含到期时间的资产)" json:"modelTypes"`
	WarningDays  int    `gorm:"column:warning_days;type:int(11);default:30;comment:剩余天数小于等于该值触发预警" json:"warningDays"`
	Severity     int8   `gorm:"column:severity;type:tinyint;default:2;comment:级别（1 紧急 2 警告 3 提醒）；已过期一律按紧急" json:"severity"`
	CooldownSec  int    `gorm:"column:cooldown_sec;type:int(11);default:86400;comment:重复提醒冷却(秒)，证书类默认一天一次" json:"cooldownSec"`
	SilenceUntil int64  `gorm:"column:silence_until;type:bigint;default:0;comment:静默截止时间(Unix秒)，0 表示不静默" json:"silenceUntil"`
	Muted        string `gorm:"column:muted;type:varchar(1);default:0;comment:静音（1 只记录事件不通知 0 正常）" json:"muted"`
	ChannelIds   string `gorm:"column:channel_ids;type:varchar(255);comment:通知渠道ID列表(逗号分隔)" json:"channelIds"`
	Enabled      string `gorm:"column:enabled;type:varchar(1);default:1;comment:启用状态（1 启用 0 停用）" json:"enabled"`
	Base
}

const TableOpsCertAlertRule = "ops_cert_alert_rule"

func (OpsCertAlertRule) TableName() string {
	return TableOpsCertAlertRule
}
