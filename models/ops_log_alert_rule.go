package models

// OpsLogAlertRule 日志告警规则：时间窗内命中条数超阈值触发，复用监控告警事件与通知链路
type OpsLogAlertRule struct {
	Name         string `gorm:"column:name;type:varchar(128);not null;comment:规则名称" json:"name"`
	Selector     string `gorm:"column:selector;type:varchar(512);comment:流选择器，如 {job=\"ops_log\",host=\"10.0.0.1\"}，为空按下方条件自动生成" json:"selector"`
	Host         string `gorm:"column:host;type:varchar(64);comment:主机过滤（空为全部主机）" json:"host"`
	Source       string `gorm:"column:source;type:varchar(64);comment:业务来源标签过滤" json:"source"`
	Keyword      string `gorm:"column:keyword;type:varchar(255);comment:关键字过滤（生成 |= 过滤器）" json:"keyword"`
	Level        string `gorm:"column:level;type:varchar(16);comment:级别过滤，如 ERROR/WARN，为空不过滤" json:"level"`
	WindowSec    int    `gorm:"column:window_sec;type:int(11);default:300;comment:统计时间窗(秒)" json:"windowSec"`
	Threshold    int    `gorm:"column:threshold;type:int(11);default:10;comment:窗口内命中条数阈值" json:"threshold"`
	Severity     int8   `gorm:"column:severity;type:tinyint;default:2;comment:级别（1 紧急 2 警告 3 提醒）" json:"severity"`
	CooldownSec  int    `gorm:"column:cooldown_sec;type:int(11);default:300;comment:重复告警冷却(秒)" json:"cooldownSec"`
	SilenceUntil int64  `gorm:"column:silence_until;type:bigint;default:0;comment:静默截止时间(Unix秒)，0 表示不静默" json:"silenceUntil"`
	Muted        string `gorm:"column:muted;type:varchar(1);default:0;comment:静音（1 只记录事件不通知 0 正常）" json:"muted"`
	ChannelIds   string `gorm:"column:channel_ids;type:varchar(255);comment:通知渠道ID列表(逗号分隔)" json:"channelIds"`
	GroupByHost  string `gorm:"column:group_by_host;type:varchar(1);default:1;comment:按主机分别告警（1 是 0 汇总）" json:"groupByHost"`
	Enabled      string `gorm:"column:enabled;type:varchar(1);default:1;comment:启用状态（1 启用 0 停用）" json:"enabled"`
	Base
}

const TableOpsLogAlertRule = "ops_log_alert_rule"

func (OpsLogAlertRule) TableName() string {
	return TableOpsLogAlertRule
}
