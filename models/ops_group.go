package models

// OpsGroup 主机分组
type OpsGroup struct {
	Name     string      `gorm:"type:varchar(32);not null;unique;comment:名称" json:"name"`
	ParentId int         `gorm:"type:int;default:0;comment:父级ID" json:"parent_id"`
	Children []*OpsGroup `gorm:"-" json:"children"` // 子级分组
	Base
}

const TableOpsGroup = "ops_group"

func (OpsGroup) TableName() string {
	return TableOpsGroup
}
