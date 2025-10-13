package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type GroupRouter struct{}

func (*GroupRouter) Setup(r *gin.RouterGroup) {
	groupController := instance.GroupController{}
	group := r.Group("/group")
	{
		group.POST("/add", groupController.AddGroupHandler)
		group.POST("/edit", groupController.EditGroupHandler)
		group.GET("/tree", groupController.GroupTreeHandler)
		group.DELETE("/rm/:id", groupController.DeleteGroupHandler)
		group.POST("/instance/ops", groupController.GroupInstanceHandler)
		group.POST("/instances/page", groupController.PageGroupInstanceHandler)
		group.POST("/instances/available", groupController.AvailableInstanceHandler)
	}
}
