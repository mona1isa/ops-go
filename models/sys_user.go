package models

import (
	"fmt"
	"gorm.io/gorm"
	"log"
	"time"
)

type SysUser struct {
	DeptId    int        `gorm:"type:bigint(20);not null;comment:部门ID" json:"deptId"`
	DeptName  string     `gorm:"-" json:"deptName"`
	UserName  string     `gorm:"type:varchar(32);not null;unique;comment:用户名" json:"userName"`
	NickName  string     `gorm:"type:varchar(32);comment:昵称" json:"nickname"`
	Email     string     `gorm:"type:varchar(64);comment:邮箱" json:"email"`
	Phone     string     `gorm:"type:varchar(16);comment:电话" json:"phone"`
	Sex       int        `gorm:"type:tinyint(4);comment:性别（0男 1女 2未知）" json:"sex"`
	Avatar    string     `gorm:"type:varchar(255);comment:头像地址" json:"avatar"`
	Password  string     `gorm:"type:varchar(512);not null;comment:密码" json:"-"`
	Status    string     `gorm:"type:varchar(1);default:0;comment:帐号状态（1正常 0停用）" json:"status"`
	LoginIP   string     `gorm:"type:varchar(128);comment:登录IP" json:"loginIP"`
	LoginDate *time.Time `gorm:"comment:登录时间" json:"loginDate"`
	RoleNames string     `gorm:"-" json:"roleNames"`
	RoleIds   []int      `gorm:"-" json:"roleIds"`
	Base
}

const TableSysUser = "sys_user"

func (SysUser) TableName() string {
	return TableSysUser
}

// AfterCreate  创建用户后，添加用户角色并同步Casbin
func (u *SysUser) AfterCreate(db *gorm.DB) error {
	// casbin 添加用户角色
	_, err := Casbin.AddUserRoles([]string{u.UserName}, u.RoleIds)
	if err != nil {
		err = fmt.Errorf("casbin添加用户角色失败：%v", err)
		return err
	}
	// 批量保存用户角色
	roleIds := u.RoleIds
	if len(roleIds) > 0 {
		for _, roleId := range roleIds {
			userRole := SysUserRole{
				UserId: u.ID,
				RoleId: roleId,
			}
			if err := DB.Create(&userRole).Error; err != nil {
				log.Println("保存用户角色失败：", err)
				return err
			}
		}
	}
	return nil
}

// BeforeUpdate 更新用户前先删除用户角色然后再创建并同步Casbin
func (u *SysUser) BeforeUpdate(db *gorm.DB) error {
	// 清除Casbin用户和角色关联
	_, err := Casbin.DeleteUserRole(u.UserName)
	if err != nil {
		err = fmt.Errorf("casbin删除用户角色失败：%v", err)
		return err
	}
	// 删除sys_user_role 中用户角色
	if err = u.deleteUserRole(); err != nil {
		return err
	}
	// 添加用户角色到 Casbin 中
	if err = u.AfterCreate(db); err != nil {
		return err
	}
	return nil
}

func (u *SysUser) deleteUserRole() error {
	if err := DB.Where("user_id = ?", u.ID).Delete(&SysUserRole{}).Error; err != nil {
		err := fmt.Errorf("删除用户角色失败：%v", err)
		return err
	}
	return nil
}
