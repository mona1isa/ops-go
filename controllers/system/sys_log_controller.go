package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/services/system"
	"net/http"
)

type SystemLogController struct {
	controllers.BaseController
}

// List 日志列表
func (c *SystemLogController) List(ctx *gin.Context) {
	logRequest := api.LogRequest{}
	if err := ctx.ShouldBindJSON(&logRequest); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err)
		return
	}
	service := system.LogService{}
	page, err := service.Page(&logRequest)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err)
		return
	}
	c.PageSuccess(ctx, page.Data, page.Total, page.TotalPage, logRequest.PageNum, logRequest.PageSize)
}
