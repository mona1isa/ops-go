package models

import "time"

type SysUser struct {
	DeptId    int        `gorm:"type:bigint(20);not null;comment:部门ID" json:"deptId"`
	UserName  string     `gorm:"type:varchar(32);not null;unique;comment:用户名" json:"userName"`
	NickName  string     `gorm:"type:varchar(32);comment:昵称" json:"nickName"`
	Email     string     `gorm:"type:varchar(64);comment:邮箱" json:"email"`
	Phone     string     `gorm:"type:varchar(16);comment:电话" json:"phone"`
	Sex       int        `gorm:"type:tinyint(4);comment:性别（0男 1女 2未知）" json:"sex"`
	Avatar    string     `gorm:"type:varchar(255);comment:头像地址" json:"avatar"`
	Password  string     `gorm:"type:varchar(512);not null;comment:密码" json:"password"`
	Status    string     `gorm:"type:varchar(1);default:0;comment:帐号状态（1正常 0停用）" json:"status"`
	LoginIP   string     `gorm:"type:varchar(128);comment:登录IP" json:"loginIP"`
	LoginDate *time.Time `gorm:"comment:登录时间" json:"loginDate"`
	Base
}

const TableSysUser = "sys_user"

func (SysUser) TableName() string {
	return TableSysUser
}
