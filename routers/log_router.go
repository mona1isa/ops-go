package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/system"
)

type LogRouter struct {
}

func (l *LogRouter) SetUp(r *gin.RouterGroup) {
	log := system.SystemLogController{}
	logGroup := r.Group("/log")
	{
		logGroup.POST("/page", log.List)
	}
}
