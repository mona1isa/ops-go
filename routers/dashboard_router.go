package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/system"
)

type DashboardRouter struct{}

func (*DashboardRouter) Setup(r *gin.RouterGroup) {
	controller := &system.DashboardController{}
	group := r.Group("/dashboard")
	{
		group.GET("/stats", controller.Stats)
	}
}
