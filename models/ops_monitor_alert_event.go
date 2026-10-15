package models

import "time"

// 告警事件状态
const (
	AlertStatusPending  = 1 // 待确认（持续时长未满足）
	AlertStatusFiring   = 2 // 告警中
	AlertStatusResolved = 3 // 已恢复
)

// OpsMonitorAlertEvent 告警事件记录
type OpsMonitorAlertEvent struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleId       int        `gorm:"index;not null;comment:规则ID" json:"ruleId"`
	RuleName     string     `gorm:"type:varchar(128);comment:规则名称（快照）" json:"ruleName"`
	Fingerprint  string     `gorm:"type:varchar(64);index;not null;comment:序列指纹（规则ID+标签集）" json:"fingerprint"`
	InstanceIp   string     `gorm:"type:varchar(32);comment:触发主机IP" json:"instanceIp"`
	Labels       string     `gorm:"type:varchar(512);comment:触发时的标签集" json:"labels"`
	Value        float64    `gorm:"type:decimal(18,4);comment:触发值" json:"value"`
	Threshold    float64    `gorm:"type:decimal(18,4);comment:阈值（快照）" json:"threshold"`
	Severity     int8       `gorm:"type:tinyint;default:2;comment:级别（1 紧急 2 警告 3 提醒）" json:"severity"`
	Status       int8       `gorm:"type:tinyint;default:1;index;comment:状态（1 待确认 2 告警中 3 已恢复）" json:"status"`
	PendingSince *time.Time `gorm:"comment:进入待确认时间" json:"pendingSince"`
	FiredAt      *time.Time `gorm:"comment:触发时间" json:"firedAt"`
	ResolvedAt   *time.Time `gorm:"comment:恢复时间" json:"resolvedAt"`
	LastNotifyAt *time.Time `gorm:"index;comment:最近通知时间（冷却判定依据）" json:"lastNotifyAt"`
	NotifyCount  int        `gorm:"type:int;default:0;comment:累计通知次数" json:"notifyCount"`
	Suppressed   int        `gorm:"type:int;default:0;comment:累计被冷却/静默抑制的次数" json:"suppressed"`
	Summary      string     `gorm:"type:varchar(512);comment:告警摘要" json:"summary"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

const TableOpsMonitorAlertEvent = "ops_monitor_alert_event"

func (OpsMonitorAlertEvent) TableName() string {
	return TableOpsMonitorAlertEvent
}
