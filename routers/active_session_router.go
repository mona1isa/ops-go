package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type ActiveSessionRouter struct{}

func (r *ActiveSessionRouter) Setup(api *gin.RouterGroup) {
	controller := &instance.ActiveSessionController{}

	group := api.Group("/active-sessions")
	{
		group.GET("/", controller.List)
		group.POST("/terminate/:sessionID", controller.Terminate)
	}
}
