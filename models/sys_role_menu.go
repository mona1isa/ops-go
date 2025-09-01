package models

type SysRoleMenu struct {
	RoleId int `gorm:"column:role_id;type:int(11);not null" json:"roleId"`
	MenuId int `gorm:"column:menu_id;type:int(11);not null" json:"menuId"`
}

const TableSysRoleMenu = "sys_role_menu"

func (SysRoleMenu) TableName() string {
	return TableSysRoleMenu
}
