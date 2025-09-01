package api

import "time"

type LoginRequest struct {
	Uuid      string     `json:"uuid" binding:"required"`     // 验证码关联uuid
	Code      string     `json:"code" binding:"required"`     // 验证码
	Username  string     `json:"username" binding:"required"` // 用户名
	Password  string     `json:"password" binding:"required"` // 密码
	LoginIP   string     `json:"loginIp"`                     // 登录IP地址
	LoginDate *time.Time `json:"loginDate"`                   // 登录时间
}
