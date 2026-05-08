package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type TaskTemplateRouter struct{}

func (*TaskTemplateRouter) Setup(r *gin.RouterGroup) {
	controller := instance.TaskTemplateController{}
	group := r.Group("/task-template")
	{
		group.POST("/page", controller.ListHandler)
		group.POST("/add", controller.AddHandler)
		group.POST("/edit", controller.EditHandler)
		group.DELETE("/rm/:id", controller.DeleteHandler)
		group.POST("/detail", controller.DetailHandler)
	}
}
