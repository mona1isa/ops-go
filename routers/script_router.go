package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type ScriptRouter struct{}

func (*ScriptRouter) Setup(r *gin.RouterGroup) {
	controller := instance.ScriptController{}
	group := r.Group("/script")
	{
		group.POST("/page", controller.ListHandler)
		group.POST("/add", controller.AddHandler)
		group.POST("/edit", controller.EditHandler)
		group.DELETE("/rm/:id", controller.DeleteHandler)
		group.GET("/info/:id", controller.DetailHandler)
	}
}
