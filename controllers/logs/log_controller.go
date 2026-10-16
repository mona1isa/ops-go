package logs

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/logs/api"
	"github.com/zhany/ops-go/models"
	logService "github.com/zhany/ops-go/services/logs"
)

// normalizePage 统一分页默认值与上限
func normalizePage(pageNum, pageSize int) (int, int) {
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return pageNum, pageSize
}

// LogQueryController 日志检索与元数据
type LogQueryController struct {
	controllers.BaseController
}

// StatusHandler 日志数据源状态
func (c *LogQueryController) StatusHandler(ctx *gin.Context) {
	service := logService.NewQueryService()
	c.Success(ctx, service.Status(ctx.Request.Context()))
}

// QueryHandler 日志明细检索
func (c *LogQueryController) QueryHandler(ctx *gin.Context) {
	request := api.LogQueryRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	service := logService.NewQueryService()
	lines, err := service.Query(ctx.Request.Context(), request)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, lines)
}

// HistogramHandler 日志时间分布
func (c *LogQueryController) HistogramHandler(ctx *gin.Context) {
	request := api.LogHistogramRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	service := logService.NewQueryService()
	points, err := service.Histogram(ctx.Request.Context(), request)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, points)
}

// ContextHandler 命中日志的上下文
func (c *LogQueryController) ContextHandler(ctx *gin.Context) {
	request := api.LogContextRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	service := logService.NewQueryService()
	result, err := service.Context(ctx.Request.Context(), request)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, result)
}

// LabelsHandler 标签名列表
func (c *LogQueryController) LabelsHandler(ctx *gin.Context) {
	service := logService.NewQueryService()
	labels, err := service.Labels(ctx.Request.Context())
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, labels)
}

// LabelValuesHandler 指定标签的取值
func (c *LogQueryController) LabelValuesHandler(ctx *gin.Context) {
	request := api.LabelsRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少标签名")
		return
	}
	service := logService.NewQueryService()
	values, err := service.LabelValues(ctx.Request.Context(), request.Label)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, values)
}

// LogSourceController 日志采集源管理
type LogSourceController struct {
	controllers.BaseController
}

// PageHandler 采集源分页
func (c *LogSourceController) PageHandler(ctx *gin.Context) {
	request := api.LogSourcePageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	pageNum, pageSize := normalizePage(request.PageNum, request.PageSize)

	service := logService.LogSourceService{}
	result, err := service.List(pageNum, pageSize, request)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AddHandler 新增采集源
func (c *LogSourceController) AddHandler(ctx *gin.Context) {
	request := api.LogSourceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	source := buildSourceModel(request)
	source.CreateBy = c.GetUserId(ctx)

	service := logService.LogSourceService{}
	if err := service.Add(&source); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// EditHandler 编辑采集源
func (c *LogSourceController) EditHandler(ctx *gin.Context) {
	request := api.LogSourceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	if request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "缺少采集源ID")
		return
	}
	service := logService.LogSourceService{}
	if err := service.Edit(request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除采集源
func (c *LogSourceController) DeleteHandler(ctx *gin.Context) {
	request := api.LogIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少采集源ID")
		return
	}
	service := logService.LogSourceService{}
	if err := service.Delete(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ToggleHandler 启停采集源
func (c *LogSourceController) ToggleHandler(ctx *gin.Context) {
	request := api.LogToggleRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	service := logService.LogSourceService{}
	if err := service.Toggle(request.Id, request.Enabled); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ResetHandler 重置读取位点
func (c *LogSourceController) ResetHandler(ctx *gin.Context) {
	request := api.LogIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少采集源ID")
		return
	}
	service := logService.LogSourceService{}
	if err := service.ResetOffset(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// CollectHandler 立即拉取一次，用于连通性与路径排障
func (c *LogSourceController) CollectHandler(ctx *gin.Context) {
	request := api.LogIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少采集源ID")
		return
	}
	result, err := logService.NewLogCollectorService().CollectNow(ctx.Request.Context(), request.Id)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, result)
}

// OptionsHandler 采集源下拉
func (c *LogSourceController) OptionsHandler(ctx *gin.Context) {
	service := logService.LogSourceService{}
	options, err := service.Options()
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, options)
}

// InstanceKeysHandler 查询主机绑定凭证
func (c *LogSourceController) InstanceKeysHandler(ctx *gin.Context) {
	request := struct {
		InstanceId int `json:"instanceId" binding:"required"`
	}{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少主机ID")
		return
	}
	service := logService.LogSourceService{}
	keys, err := service.InstanceKeys(request.InstanceId)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, keys)
}

// buildSourceModel 请求体转模型，新增场景下位点从零开始
func buildSourceModel(request api.LogSourceRequest) models.OpsLogSource {
	return models.OpsLogSource{
		Name: request.Name, InstanceId: request.InstanceId, KeyId: request.KeyId,
		Path: request.Path, Job: request.Job, Source: request.Source, Labels: request.Labels,
		LevelRegex: request.LevelRegex, Enabled: request.Enabled,
		IntervalSec: request.IntervalSec, TimeoutSec: request.TimeoutSec,
		MaxBytes: request.MaxBytes, BackBytes: request.BackBytes,
		Base: models.Base{Remark: request.Remark},
	}
}
