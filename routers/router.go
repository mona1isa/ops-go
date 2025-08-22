package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/middleware"
)

func Init() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.LogMiddleware())

	api := r.Group("/api")
	userRouter := &UserRouter{}
	userRouter.Setup(api)
	return r
}
