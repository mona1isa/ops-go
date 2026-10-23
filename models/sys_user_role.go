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
// 列名必须与查询里的 snake_case 别名一致：PostgreSQL 会把未加引号的标识符折叠成小写，
// 用 camelCase 别名（AS roleName）时驱动返回 rolename，GORM 匹配不上字段，Scan 结果全为零值。
type SysUserRoleResult struct {
	UserId   int    `gorm:"column:user_id" json:"userId"`
	RoleName string `gorm:"column:role_name" json:"roleName"`
	RoleId   int    `gorm:"column:role_id" json:"roleId"`
}
