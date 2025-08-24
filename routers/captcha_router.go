package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/system"
)

type CaptchaRouter struct{}

func (*CaptchaRouter) Setup(r *gin.RouterGroup) {
	c := &system.CaptchaController{}
	captchaGroup := r.Group("/captcha")
	{
		captchaGroup.POST("/generate", c.GenerateCaptchaHandler)
		captchaGroup.POST("/verify", c.VerifyCaptchaHandler)
	}
}
