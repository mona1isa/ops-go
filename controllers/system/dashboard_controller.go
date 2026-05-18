package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/services/system"
	"net/http"
)

type DashboardController struct {
	controllers.BaseController
}

// Stats 获取首页仪表盘统计数据
func (c *DashboardController) Stats(ctx *gin.Context) {
	service := &system.DashboardService{}
	stats, err := service.GetStats()
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, stats)
}
