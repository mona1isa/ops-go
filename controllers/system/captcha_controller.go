package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/services/system"
	"net/http"
	"os"
	"strings"
)

type CaptchaController struct {
}

// GenerateCaptchaHandler 生成验证码
func (c *CaptchaController) GenerateCaptchaHandler(ctx *gin.Context) {
	// 检查验证码开关
	captchaEnabled := isCaptchaEnabled()

	if !captchaEnabled {
		ctx.JSON(http.StatusOK, gin.H{
			"code":           http.StatusOK,
			"msg":            "success",
			"captchaEnabled": false,
		})
		return
	}

	service := system.CaptchaService{}
	captcha := service.GenerateCaptcha()
	if captcha == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"msg":  "生成验证码异常",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":           http.StatusOK,
		"msg":            "success",
		"uuid":           captcha.Uuid,
		"img":            captcha.Img,
		"captchaEnabled": true,
	})
}

// isCaptchaEnabled 检查验证码开关是否开启
func isCaptchaEnabled() bool {
	val := os.Getenv("CAPTCHA_ENABLED")
	return strings.EqualFold(val, "true") || val == "1"
}
