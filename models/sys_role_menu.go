package models

type SysRoleMenu struct {
	RoleId int `gorm:"type:int(11);not null" json:"roleId"`
	MenuId int `gorm:"type:int(11);not null" json:"menuId"`
}

const TableSysRoleMenu = "sys_role_menu"

func (SysRoleMenu) TableName() string {
	return TableSysRoleMenu
}
