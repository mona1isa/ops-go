package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type TaskExecutionRouter struct{}

func (*TaskExecutionRouter) Setup(r *gin.RouterGroup) {
	controller := instance.TaskExecutionController{}
	group := r.Group("/task-execution")
	{
		group.POST("/execute", controller.QuickExecuteHandler)
		group.POST("/execute-template", controller.TemplateExecuteHandler)
		group.POST("/execute-pipeline", controller.PipelineExecuteHandler)
		group.POST("/cancel", controller.CancelHandler)
		group.POST("/page", controller.ListHandler)
		group.POST("/detail", controller.DetailHandler)
		group.POST("/host-result", controller.HostResultHandler)
	}
}
