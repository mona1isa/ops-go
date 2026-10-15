package models

// OpsMonitorRule 告警规则
type OpsMonitorRule struct {
	Name         string  `gorm:"column:name;type:varchar(128);not null;comment:规则名称" json:"name"`
	Metric       string  `gorm:"column:metric;type:varchar(64);comment:指标名称（展示用）" json:"metric"`
	PromQL       string  `gorm:"column:prom_ql;type:varchar(512);not null;comment:PromQL 表达式，可用 {{instance}} 占位按主机展开" json:"promQl"`
	Operator     string  `gorm:"column:operator;type:varchar(4);default:'>';comment:比较符（> >= < <= == !=），PromQL 未写比较时生效" json:"operator"`
	Threshold    float64 `gorm:"column:threshold;type:decimal(18,4);default:0;comment:阈值" json:"threshold"`
	Severity     int8    `gorm:"column:severity;type:tinyint;default:2;comment:级别（1 紧急 2 警告 3 提醒）" json:"severity"`
	ForSec       int     `gorm:"column:for_sec;type:int(11);default:0;comment:持续时长(秒)后才触发" json:"forSec"`
	CooldownSec  int     `gorm:"column:cooldown_sec;type:int(11);default:300;comment:重复告警冷却(秒)" json:"cooldownSec"`
	SilenceUntil int64   `gorm:"column:silence_until;type:bigint;default:0;comment:静默截止时间(Unix秒)，0 表示不静默" json:"silenceUntil"`
	Muted        string  `gorm:"column:muted;type:varchar(1);default:0;comment:静音（1 只记录事件不通知 0 正常）" json:"muted"`
	ChannelIds   string  `gorm:"column:channel_ids;type:varchar(255);comment:通知渠道ID列表(逗号分隔)" json:"channelIds"`
	Enabled      string  `gorm:"column:enabled;type:varchar(1);default:1;comment:启用状态（1 启用 0 停用）" json:"enabled"`
	Base
}

const TableOpsMonitorRule = "ops_monitor_rule"

func (OpsMonitorRule) TableName() string {
	return TableOpsMonitorRule
}

// 告警级别
const (
	SeverityCritical = 1 // 紧急
	SeverityWarning  = 2 // 警告
	SeverityInfo     = 3 // 提醒
)
