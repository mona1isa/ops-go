package models

import "time"

type SysUserToken struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId    int       `gorm:"userId" json:"userId"`            // 用户ID
	Token     string    `gorm:"token" json:"token"`              // 用户Token
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"` // 创建时间
	ExpireAt  time.Time `gorm:"expireAt" json:"expireAt"`        // 过期时间
}

const TableSysUserToken = "sys_user_token"

func (SysUserToken) TableName() string {
	return TableSysUserToken
}
