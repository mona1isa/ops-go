package system

import (
	"fmt"
	"github.com/google/uuid"
)

type CaptchaService struct {
}

// GenerateCaptcha 生成验证码
func (c *CaptchaService) GenerateCaptcha() {
	// 生成uuid
	uuidV4 := uuid.New().String()
	fmt.Println(uuidV4)
}
