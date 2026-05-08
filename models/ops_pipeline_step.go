package models

// 失败策略
const (
	OnFailureAbort  = 1 // 终止
	OnFailureSkip   = 2 // 跳过继续
	OnFailureRetry  = 3 // 重试
)

// OpsPipelineStep 编排步骤
type OpsPipelineStep struct {
	PipelineId   int    `gorm:"type:int;not null;index;comment:编排ID" json:"pipelineId"`
	StepName     string `gorm:"type:varchar(128);not null;comment:步骤名称" json:"stepName"`
	TemplateId   int    `gorm:"type:int;not null;comment:任务模板ID" json:"templateId"`
	StepOrder    int    `gorm:"type:int;default:0;comment:执行顺序" json:"stepOrder"`
	ParentStepId int    `gorm:"type:int;default:0;comment:父步骤ID 0=顶层" json:"parentStepId"`
	Condition    string `gorm:"type:varchar(255);comment:执行条件" json:"condition"`
	OnFailure   int8   `gorm:"type:tinyint;default:1;comment:失败策略 1终止 2跳过 3重试" json:"onFailure"`
	RetryCount   int    `gorm:"type:int;default:0;comment:重试次数" json:"retryCount"`
	Base
}
