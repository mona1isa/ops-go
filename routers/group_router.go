package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type GroupRouter struct{}

func (*GroupRouter) Setup(r *gin.RouterGroup) {
	groupController := instance.GroupController{}
	hostScanController := instance.HostScanController{}
	group := r.Group("/group")
	{
		group.POST("/add", groupController.AddGroupHandler)
		group.POST("/edit", groupController.EditGroupHandler)
		group.GET("/tree", groupController.GroupTreeHandler)
		group.DELETE("/rm/:id", groupController.DeleteGroupHandler)
		group.POST("/instance/ops", groupController.GroupInstanceHandler)
		group.POST("/instances/page", groupController.PageGroupInstanceHandler)
		group.POST("/instances/available", groupController.AvailableInstanceHandler)
		// 同步主机相关接口
		group.POST("/sync/scan", hostScanController.ScanHostsHandler)
		group.POST("/sync/save", hostScanController.SaveScannedHostsHandler)
	}
}
