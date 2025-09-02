package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/middleware"
)

func Init() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.Cors())
	r.Use(middleware.LogMiddleware())
	r.Use(middleware.AuthMiddleware())

	api := r.Group("/api")
	userRouter := &UserRouter{}
	captchaRouter := &CaptchaRouter{}
	logRouter := LogRouter{}
	roleRouter := RoleRouter{}
	menuRouter := MenuRouter{}

	userRouter.Setup(api)
	captchaRouter.Setup(api)
	logRouter.SetUp(api)
	roleRouter.Setup(api)
	menuRouter.Setup(api)
	return r
}
