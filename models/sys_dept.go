package models

type SysDept struct {
	Name     string `gorm:"type:varchar(32);not null;unique;comment:部门名称" json:"name"`
	ParentId int    `gorm:"type:bigint(20);comment:父级ID" json:"parentId"`
	OrderNum int    `gorm:"type:int(4);comment:排序" json:"orderNum"`
	Status   bool   `gorm:"type:tinyint(1);default:0;comment:状态（true正常 false停用）" json:"status"`
	Base
}

const TableSysDept = "sys_dept"

func (SysDept) TableName() string {
	return TableSysDept
}
