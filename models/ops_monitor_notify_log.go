package models

import "time"

// OpsMonitorNotifyLog 告警通知发送记录
type OpsMonitorNotifyLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	EventId      uint64    `gorm:"index;comment:告警事件ID" json:"eventId"`
	RuleId       int       `gorm:"index;comment:规则ID" json:"ruleId"`
	ChannelId    int       `gorm:"index;comment:渠道ID" json:"channelId"`
	ChannelName  string    `gorm:"type:varchar(128);comment:渠道名称（快照）" json:"channelName"`
	Recovered    int8      `gorm:"type:tinyint;default:0;comment:事件类型（0 告警触发 1 告警恢复）" json:"recovered"`
	Payload      string    `gorm:"type:text;comment:请求体" json:"payload"`
	StatusCode   int       `gorm:"type:int;comment:HTTP 响应码" json:"statusCode"`
	ResponseBody string    `gorm:"type:varchar(1024);comment:响应体（截断）" json:"responseBody"`
	Attempt      int       `gorm:"type:int;default:1;comment:第几次尝试" json:"attempt"`
	Success      int8      `gorm:"type:tinyint;default:0;comment:是否成功（1 成功 0 失败）" json:"success"`
	ErrorMsg     string    `gorm:"type:varchar(512);comment:失败原因" json:"errorMsg"`
	CreatedAt    time.Time `json:"createdAt"`
}

const TableOpsMonitorNotifyLog = "ops_monitor_notify_log"

func (OpsMonitorNotifyLog) TableName() string {
	return TableOpsMonitorNotifyLog
}
