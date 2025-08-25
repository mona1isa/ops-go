package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/services/system"
	"net/http"
)

type CaptchaController struct {
}

// GenerateCaptchaHandler 生成验证码
func (c *CaptchaController) GenerateCaptchaHandler(ctx *gin.Context) {
	service := system.CaptchaService{}
	captcha := service.GenerateCaptcha()
	if captcha == nil {
		result := gin.H{
			"code": http.StatusInternalServerError,
			"msg":  "生成验证码异常",
		}
		ctx.JSON(http.StatusInternalServerError, result)
	}

	result := gin.H{
		"code": http.StatusOK,
		"msg":  "success",
		"uuid": captcha.Uuid,
		"img":  captcha.Img,
	}
	ctx.JSON(http.StatusOK, result)
}
