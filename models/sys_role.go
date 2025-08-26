package models

type SysRole struct {
	Base
	Name     string `json:"name" gorm:"column:name;type:varchar(255);not null;comment:角色名称"`
	OrderNum int    `json:"orderNum" gorm:"column:order_num;type:int(2);default:1;comment:排序"`
	Status   string `json:"status" gorm:"column:status;type:varchar(1);default:1;comment:状态（0：禁用，1：正常）"`
}

const TableSysRole = "sys_role"

func (SysRole) TableName() string {
	return TableSysRole
}
