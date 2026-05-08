package models

import "time"

// 步骤执行状态
const (
	StepStatusPending  = 1 // 待执行
	StepStatusRunning  = 2 // 执行中
	StepStatusSuccess  = 3 // 成功
	StepStatusFail     = 4 // 失败
	StepStatusSkipped  = 5 // 跳过
)

// OpsStepExecution 编排步骤执行
type OpsStepExecution struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ExecutionId uint64    `gorm:"index;not null;comment:执行记录ID" json:"executionId"`
	StepId     int        `gorm:"not null;comment:编排步骤ID" json:"stepId"`
	StepName   string     `gorm:"type:varchar(128);not null;comment:步骤名称" json:"stepName"`
	TemplateId int        `gorm:"not null;comment:任务模板ID" json:"templateId"`
	Status     int8       `gorm:"type:tinyint;default:1;comment:状态 1待执行 2执行中 3成功 4失败 5跳过" json:"status"`
	StartedAt  *time.Time `gorm:"comment:开始时间" json:"startedAt"`
	FinishedAt *time.Time `gorm:"comment:结束时间" json:"finishedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}
