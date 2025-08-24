package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/services/system"
)

type CaptchaController struct {
}

// GenerateCaptchaHandler 生成验证码
func (c *CaptchaController) GenerateCaptchaHandler(ctx *gin.Context) {
	service := system.CaptchaService{}
	service.GenerateCaptcha()
}

// VerifyCaptchaHandler 验证验证码
func (c *CaptchaController) VerifyCaptchaHandler(ctx *gin.Context) {

}
