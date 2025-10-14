package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type KeysRouter struct{}

func (*KeysRouter) Setup(r *gin.RouterGroup) {
	keysController := instance.KeysController{}
	keysGroup := r.Group("/keys")
	{
		keysGroup.GET("/list", keysController.ListHandler)
		keysGroup.POST("/add", keysController.AddKeyHandler)
		keysGroup.POST("/edit", keysController.EditKeyHandler)
		keysGroup.POST("/page", keysController.PageKeyHandler)
		keysGroup.POST("/changeStatus", keysController.ChangeStatusHandler)
		keysGroup.DELETE("/rm/:id", keysController.DeleteKeyHandler)
		keysGroup.GET("/available/list/:instanceId", keysController.AvailableKeysHandler)
		keysGroup.POST("/available/listBy/osType", keysController.AvailableKeysBySystemHandler)
	}
}
