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

// AfterCreate 创建用户后，添加用户角色并同步Casbin
func (u *SysUser) AfterCreate(db *gorm.DB) error {
	roleIds := u.RoleIds
	if len(roleIds) == 0 {
		return nil
	}

	// 先检查 Casbin 是否初始化
	if !Casbin.IsInitialized() {
		log.Printf("Casbin 未初始化，跳过同步用户角色到 Casbin: %s", u.UserName)
		// 只保存数据库关系，不返回错误
	} else {
		// Casbin 添加用户角色
		_, err := Casbin.AddUserRoles([]string{u.UserName}, roleIds)
		if err != nil {
			log.Printf("casbin添加用户角色失败：%v", err)
			// 不返回错误，允许用户创建成功
		}
	}

	// 批量保存用户角色到数据库
	userRoles := make([]SysUserRole, 0, len(roleIds))
	for _, roleId := range roleIds {
		userRoles = append(userRoles, SysUserRole{
			UserId: u.ID,
			RoleId: roleId,
		})
	}

	if len(userRoles) > 0 {
		if err := db.Create(&userRoles).Error; err != nil {
			log.Println("保存用户角色失败：", err)
			return fmt.Errorf("保存用户角色失败：%v", err)
		}
	}

	return nil
}

// BeforeUpdate 更新用户前先删除用户角色然后再创建并同步Casbin
func (u *SysUser) BeforeUpdate(db *gorm.DB) error {
	// 只有当 RoleIds 字段在更新中存在时才同步角色
	// 注意：GORM 的 Changed 方法检查的是结构体字段是否为零值
	// 由于 RoleIds 是切片类型，我们需要特殊处理
	if len(u.RoleIds) == 0 {
		// 检查是否是明确的清空操作
		// 通过检查 UpdateColumn 或其他方式确定
		return nil
	}

	// 在事务中执行
	return db.Transaction(func(tx *gorm.DB) error {
		// 清除 Casbin 用户和角色关联（如果已初始化）
		if Casbin.IsInitialized() {
			_, err := Casbin.DeleteUserRole(u.UserName)
			if err != nil {
				log.Printf("casbin删除用户角色失败：%v\n", err)
				// 不返回错误，继续执行
			}
		}

		// 删除 sys_user_role 中用户角色
		if err := tx.Where("user_id = ?", u.ID).Delete(&SysUserRole{}).Error; err != nil {
			return fmt.Errorf("删除用户角色失败：%v", err)
		}

		// 批量保存新的用户角色
		userRoles := make([]SysUserRole, 0, len(u.RoleIds))
		for _, roleId := range u.RoleIds {
			userRoles = append(userRoles, SysUserRole{
				UserId: u.ID,
				RoleId: roleId,
			})
		}

		if len(userRoles) > 0 {
			if err := tx.Create(&userRoles).Error; err != nil {
				return fmt.Errorf("保存用户角色失败：%v", err)
			}
		}

		// 同步到 Casbin（如果已初始化）
		if Casbin.IsInitialized() {
			_, err := Casbin.AddUserRoles([]string{u.UserName}, u.RoleIds)
			if err != nil {
				log.Printf("casbin添加用户角色失败：%v\n", err)
				// 不返回错误，允许更新成功
			}
		}

		return nil
	})
}

func (u *SysUser) AfterDelete(db *gorm.DB) error {
	// 在事务中执行删除操作
	return db.Transaction(func(tx *gorm.DB) error {
		// 清理 Casbin 用户和角色关联（如果已初始化）
		if Casbin.IsInitialized() {
			_, err := Casbin.DeleteUserRole(u.UserName)
			if err != nil {
				log.Printf("casbin删除用户角色失败：%v\n", err)
				// 不返回错误，继续删除数据库记录
			}
		}

		// 删除用户角色
		if err := tx.Where("user_id = ?", u.ID).Delete(&SysUserRole{}).Error; err != nil {
			return fmt.Errorf("删除用户角色失败：%v", err)
		}

		// 删除用户 Token
		if err := tx.Where("user_id = ?", u.ID).Delete(&SysUserToken{}).Error; err != nil {
			log.Printf("删除用户Token失败：%v\n", err)
			// 不返回错误，继续执行
		}

		return nil
	})
}

// UpdateLoginInfo 只更新部分信息不需要执行 hook 函数时使用
func (u *SysUser) UpdateLoginInfo() {
	user := SysUser{
		LoginIP:   u.LoginIP,
		LoginDate: u.LoginDate,
	}
	if err := DB.Model(&SysUser{}).Where("id = ?", u.ID).UpdateColumns(&user).Error; err != nil {
		log.Println("更新用户登录信息失败：", err)
	}
}
