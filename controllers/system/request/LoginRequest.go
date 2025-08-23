package request

type LoginRequest struct {
	Uuid     string `json:"uuid"`     // 验证码关联uuid
	Code     string `json:"code"`     // 验证码
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码
}
