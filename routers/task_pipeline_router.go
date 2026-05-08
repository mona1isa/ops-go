package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type TaskPipelineRouter struct{}

func (*TaskPipelineRouter) Setup(r *gin.RouterGroup) {
	controller := instance.TaskPipelineController{}
	group := r.Group("/task-pipeline")
	{
		group.POST("/page", controller.ListHandler)
		group.POST("/add", controller.AddHandler)
		group.POST("/edit", controller.EditHandler)
		group.DELETE("/rm/:id", controller.DeleteHandler)
		group.POST("/detail", controller.DetailHandler)
	}
}
