package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/middleware"
)

func Init() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.LogMiddleware())
	r.Use(middleware.AuthMiddleware())
	r.Use(middleware.CasbinMiddleware())

	api := r.Group("/api")
	userRouter := &UserRouter{}
	captchaRouter := &CaptchaRouter{}
	logRouter := LogRouter{}
	roleRouter := RoleRouter{}
	menuRouter := MenuRouter{}
	deptRouter := DeptRouter{}
	instanceRouter := InstanceRouter{}
	keysRouter := KeysRouter{}
	groupRouter := GroupRouter{}
	userInstanceAuthRouter := UserInstanceAuthRouter{}
	sessionRecordRouter := &SessionRecordRouter{}
	activeSessionRouter := &ActiveSessionRouter{}
	dangerousCommandRouter := &DangerousCommandRouter{}

	userRouter.Setup(api)
	captchaRouter.Setup(api)
	logRouter.SetUp(api)
	roleRouter.Setup(api)
	menuRouter.Setup(api)
	deptRouter.Setup(api)
	instanceRouter.Setup(api)
	keysRouter.Setup(api)
	groupRouter.Setup(api)
	userInstanceAuthRouter.Setup(api)
	sessionRecordRouter.Setup(api)
	activeSessionRouter.Setup(api)
	dangerousCommandRouter.Setup(api)
	return r
}
