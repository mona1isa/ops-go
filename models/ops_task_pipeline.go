package models

// OpsTaskPipeline 任务编排
type OpsTaskPipeline struct {
	Name        string `gorm:"type:varchar(128);not null;comment:编排名称" json:"name"`
	Description string `gorm:"type:varchar(255);comment:描述" json:"description"`
	Steps       []OpsPipelineStep `gorm:"foreignKey:PipelineId" json:"steps"`
	Base
}
