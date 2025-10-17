package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type UserInstanceAuthRouter struct{}

func (*UserInstanceAuthRouter) Setup(r *gin.RouterGroup) {
	userInstanceAuthController := instance.UserInstanceAuthController{}
	group := r.Group("/user/instance/auth")
	{
		group.POST("/add", userInstanceAuthController.AddHandler)
		group.POST("/delete", userInstanceAuthController.DeleteHandler)
		group.POST("/list", userInstanceAuthController.UserInstanceAuthHandler)
		group.POST("/listInstance", userInstanceAuthController.ListInstanceHandler)
		group.POST("/pageUserInstances", userInstanceAuthController.PageUserInstancesHandler)
		group.POST("/pageUserGroups", userInstanceAuthController.PageUserGroupHandler)
		group.POST("/available/instances", userInstanceAuthController.AvailableInstancesHandler)
		group.POST("/available/groups", userInstanceAuthController.AvailableGroupsHandler)
		group.POST("/available/keys", userInstanceAuthController.AvailableKeysHandler)
	}
}
