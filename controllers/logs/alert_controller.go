package logs

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/logs/api"
	monitorApi "github.com/zhany/ops-go/controllers/monitor/api"
	"github.com/zhany/ops-go/models"
	logService "github.com/zhany/ops-go/services/logs"
	monitorService "github.com/zhany/ops-go/services/monitor"
)

// LogAlertController 日志告警规则与事件
type LogAlertController struct {
	controllers.BaseController
}

// PageHandler 规则分页
func (c *LogAlertController) PageHandler(ctx *gin.Context) {
	request := api.LogAlertPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	pageNum, pageSize := normalizePage(request.PageNum, request.PageSize)

	service := logService.LogAlertService{}
	result, err := service.List(pageNum, pageSize, request)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AddHandler 新增规则
func (c *LogAlertController) AddHandler(ctx *gin.Context) {
	request := api.LogAlertRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	rule := models.OpsLogAlertRule{
		Name: request.Name, Selector: request.Selector, Host: request.Host, Source: request.Source,
		Keyword: request.Keyword, Level: request.Level, WindowSec: request.WindowSec,
		Threshold: request.Threshold, Severity: request.Severity, CooldownSec: request.CooldownSec,
		Muted: request.Muted, ChannelIds: request.ChannelIds, GroupByHost: request.GroupByHost,
		Enabled: request.Enabled,
		Base:    models.Base{Remark: request.Remark},
	}
	rule.CreateBy = c.GetUserId(ctx)

	service := logService.LogAlertService{}
	if err := service.Add(&rule); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// EditHandler 编辑规则
func (c *LogAlertController) EditHandler(ctx *gin.Context) {
	request := api.LogAlertRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	if request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "缺少规则ID")
		return
	}
	service := logService.LogAlertService{}
	if err := service.Edit(request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除规则
func (c *LogAlertController) DeleteHandler(ctx *gin.Context) {
	request := api.LogIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少规则ID")
		return
	}
	service := logService.LogAlertService{}
	if err := service.Delete(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ToggleHandler 启停规则
func (c *LogAlertController) ToggleHandler(ctx *gin.Context) {
	request := api.LogToggleRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	service := logService.LogAlertService{}
	if err := service.Toggle(request.Id, request.Enabled); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// SilenceHandler 静默规则，minute<=0 表示立即解除
func (c *LogAlertController) SilenceHandler(ctx *gin.Context) {
	request := api.LogAlertSilenceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	service := logService.LogAlertService{}
	if err := service.Silence(request.Id, request.Minute); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// TestHandler 试算：按当前条件查一次窗口计数，用于配置规则时验证效果
func (c *LogAlertController) TestHandler(ctx *gin.Context) {
	request := api.LogAlertRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	service := logService.LogAlertService{}
	result, err := service.Test(ctx.Request.Context(), request)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.Success(ctx, result)
}

// EventsHandler 日志告警事件列表，复用监控告警事件表，按来源过滤
func (c *LogAlertController) EventsHandler(ctx *gin.Context) {
	request := monitorApi.EventPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	pageNum, pageSize := normalizePage(request.PageNum, request.PageSize)

	service := monitorService.EventService{}
	result, err := service.List(pageNum, pageSize, monitorService.EventFilter{
		RuleId:   request.RuleId,
		Severity: request.Severity,
		Status:   request.Status,
		Source:   models.AlertSourceLog,
		StartAt:  request.StartAt,
		EndAt:    request.EndAt,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}
