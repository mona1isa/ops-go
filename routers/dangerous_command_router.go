package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type DangerousCommandRouter struct{}

func (*DangerousCommandRouter) Setup(r *gin.RouterGroup) {
	controller := instance.DangerousCommandController{}
	group := r.Group("/dangerousCommand")
	{
		group.POST("/list", controller.ListHandler)
		group.POST("/add", controller.AddHandler)
		group.POST("/edit", controller.EditHandler)
		group.DELETE("/rm/:id", controller.DeleteHandler)
		group.POST("/changeStatus", controller.ChangeStatusHandler)
	}
}
