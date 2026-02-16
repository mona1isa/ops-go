package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type SessionRecordRouter struct{}

func (r *SessionRecordRouter) Setup(api *gin.RouterGroup) {
	controller := &instance.SessionRecordController{}

	group := api.Group("/session-record")
	{
		// 注意：特定路径必须在通配路由 /:id 之前注册，否则会被错误匹配
		group.GET("/list", controller.List)
		group.GET("/statistics", controller.Statistics)
		group.GET("/playback/:id", controller.Playback)
		group.GET("/download/:id", controller.Download)
		group.GET("/:id", controller.Get)
		group.DELETE("/:id", controller.Delete)
	}
}
