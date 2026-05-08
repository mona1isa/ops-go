package models

import "time"

// 主机执行状态
const (
	HostStatusPending  = 1 // 待执行
	HostStatusRunning  = 2 // 执行中
	HostStatusSuccess  = 3 // 成功
	HostStatusFail     = 4 // 失败
	HostStatusTimeout  = 5 // 超时
	HostStatusSkipped  = 6 // 跳过
)

// OpsExecutionHost 执行-主机关联
type OpsExecutionHost struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ExecutionId  uint64     `gorm:"index;not null;comment:执行记录ID" json:"executionId"`
	StepExecId   uint64     `gorm:"type:int;default:0;comment:编排步骤执行ID 0=非编排" json:"stepExecId"`
	InstanceId   int        `gorm:"index;not null;comment:主机ID" json:"instanceId"`
	InstanceName string     `gorm:"type:varchar(32);not null;comment:主机名称" json:"instanceName"`
	InstanceIP   string     `gorm:"type:varchar(32);not null;comment:主机IP" json:"instanceIp"`
	KeyId        int        `gorm:"not null;comment:凭证ID" json:"keyId"`
	KeyName      string     `gorm:"type:varchar(32);not null;comment:凭证名称" json:"keyName"`
	KeyUser      string     `gorm:"type:varchar(32);not null;comment:登录用户" json:"keyUser"`
	Status       int8       `gorm:"type:tinyint;default:1;comment:状态 1待执行 2执行中 3成功 4失败 5超时 6跳过" json:"status"`
	Result       string     `gorm:"type:text;comment:执行结果输出" json:"result"`
	ErrorMsg     string     `gorm:"type:text;comment:错误信息" json:"errorMsg"`
	StartedAt    *time.Time `gorm:"comment:开始时间" json:"startedAt"`
	FinishedAt   *time.Time `gorm:"comment:结束时间" json:"finishedAt"`
	Duration     int        `gorm:"type:int;default:0;comment:执行耗时(毫秒)" json:"duration"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}
