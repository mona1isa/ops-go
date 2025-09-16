package models

type SysUserRole struct {
	UserId int `gorm:"type:bigint(20);not null;comment:用户ID" json:"userId"`
	RoleId int `gorm:"type:bigint(20);not null;comment:角色ID" json:"roleId"`
}

const TableSysUserRole = "sys_user_role"

func (SysUserRole) TableName() string {
	return TableSysUserRole
}

// SysUserRoleResult 用户角色查询结果集
type SysUserRoleResult struct {
	UserID   int    `gorm:"column:userId" json:"userId"`
	RoleName string `gorm:"column:roleName" json:"roleName"`
	RoleID   int    `gorm:"column:roleId" json:"roleId"`
}
