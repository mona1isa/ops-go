package api

import "time"

type LoginRequest struct {
	Uuid      string     `json:"uuid" binding:"omitempty"`    // 验证码关联uuid
	Code      string     `json:"code" binding:"omitempty"`    // 验证码
	Username  string     `json:"username" binding:"required"` // 用户名
	Password  string     `json:"password" binding:"required"` // 密码
	LoginIP   string     `json:"loginIp"`                     // 登录IP地址
	LoginDate *time.Time `json:"loginDate"`                   // 登录时间
	UserAgent string     `json:"-"`                           // 客户端UA，由控制层注入，用于登录日志留痕
}

// LoginResult 登录第一步（密码校验）的分流结果：
// Token 非空表示可直接完成登录；MfaRequired 表示需要提交动态码二次验证；
// BindRequired 表示系统强制 MFA 但该用户尚未绑定（前端登录成功后引导绑定）
type LoginResult struct {
	Token        string `json:"token,omitempty"`
	MfaRequired  bool   `json:"mfaRequired,omitempty"`
	PreToken     string `json:"preToken,omitempty"`
	BindRequired bool   `json:"bindRequired,omitempty"`
}

// MfaLoginRequest 登录第二步：动态验证码或一次性恢复码
type MfaLoginRequest struct {
	PreToken string `json:"preToken" binding:"required"` // 第一步返回的一次性票据
	Code     string `json:"code"`                        // 6 位 TOTP 动态码
	Recovery string `json:"recovery"`                    // 恢复码（与 Code 二选一）
}

// MfaConfirmRequest 绑定确认：输入认证器生成的当前口令
type MfaConfirmRequest struct {
	Code string `json:"code" binding:"required"`
}

// MfaUnbindRequest 解绑：需验证当前密码（RSA 加密 + base64，与修改密码一致）
type MfaUnbindRequest struct {
	Password string `json:"password" binding:"required"`
}
