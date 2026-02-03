package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type InstanceRouter struct {
}

func (*InstanceRouter) Setup(r *gin.RouterGroup) {
	instanceController := instance.InstanceController{}
	wsController := instance.WebSocketController{}
	instanceGroup := r.Group("/instance")
	{
		instanceGroup.POST("/add", instanceController.AddInstanceHandler)
		instanceGroup.POST("/edit", instanceController.EditInstanceHandler)
		instanceGroup.POST("/changeStatus", instanceController.ChangeStatus)
		instanceGroup.POST("/page", instanceController.PageInstanceHandler)
		instanceGroup.GET("/info/:id", instanceController.GetInstanceDetailHandler)
		instanceGroup.DELETE("/rm/:id", instanceController.DeleteInstanceHandler)
		instanceGroup.POST("/keys/binding", instanceController.KeyBindingHandler)
		instanceGroup.POST("/keys/unbinding", instanceController.UnBindingKeyHandler)
		instanceGroup.POST("/keys/testConnect", instanceController.TestConnectHandler)

		instanceGroup.POST("/myInstance", instanceController.GetMyInstanceHandler)

		// WebSocket终端连接接口
		instanceGroup.GET("/terminal", wsController.WebSocketHandler)
	}
}
