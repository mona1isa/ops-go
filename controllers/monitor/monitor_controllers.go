package monitor

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	api "github.com/zhany/ops-go/controllers/monitor/api"
	"github.com/zhany/ops-go/models"
	monitorService "github.com/zhany/ops-go/services/monitor"
)

// RuleController 告警规则管理
type RuleController struct {
	controllers.BaseController
}

// PageHandler 分页查询告警规则
func (c *RuleController) PageHandler(ctx *gin.Context) {
	request := api.RulePageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	pageNum, pageSize := normalizePage(request.PageNum, request.PageSize)

	service := monitorService.RuleService{}
	result, err := service.List(pageNum, pageSize, monitorService.RuleFilter{
		Name:     request.Name,
		Severity: request.Severity,
		Muted:    request.Muted,
		Enabled:  request.Enabled,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AddHandler 新增告警规则
func (c *RuleController) AddHandler(ctx *gin.Context) {
	request := api.RuleRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：规则名称与查询表达式不能为空")
		return
	}
	userId := c.GetUserId(ctx)
	rule := buildRule(&request, userId)

	service := monitorService.RuleService{}
	if err := service.Add(rule); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// EditHandler 编辑告警规则
func (c *RuleController) EditHandler(ctx *gin.Context) {
	request := api.RuleRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：规则名称与查询表达式不能为空")
		return
	}
	if request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少规则ID")
		return
	}
	userId := c.GetUserId(ctx)
	rule := buildRule(&request, userId)
	rule.ID = request.Id

	service := monitorService.RuleService{}
	if err := service.Edit(rule); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除告警规则
func (c *RuleController) DeleteHandler(ctx *gin.Context) {
	request := api.RuleIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少规则ID")
		return
	}
	service := monitorService.RuleService{}
	if err := service.Delete(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ToggleHandler 启停告警规则
func (c *RuleController) ToggleHandler(ctx *gin.Context) {
	request := api.RuleIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少规则ID")
		return
	}
	service := monitorService.RuleService{}
	if err := service.Toggle(request.Id, request.Enabled); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// SilenceHandler 设置规则静默
func (c *RuleController) SilenceHandler(ctx *gin.Context) {
	request := api.RuleSilenceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少规则ID")
		return
	}
	seconds := request.Seconds
	if request.Disabled {
		seconds = 0
	}
	service := monitorService.RuleService{}
	if err := service.Silence(request.Id, seconds); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// TestHandler 规则试算，即时查询数据源但不产生事件与通知
func (c *RuleController) TestHandler(ctx *gin.Context) {
	request := api.RuleTestRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少查询表达式")
		return
	}
	service := monitorService.RuleService{}
	result, err := service.Test(request.PromQl, request.Operator, request.Threshold)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, result)
}

func buildRule(request *api.RuleRequest, userId string) *models.OpsMonitorRule {
	return &models.OpsMonitorRule{
		Name:         request.Name,
		Metric:       request.Metric,
		PromQL:       request.PromQl,
		Operator:     request.Operator,
		Threshold:    request.Threshold,
		Severity:     request.Severity,
		ForSec:       request.ForSec,
		CooldownSec:  request.CooldownSec,
		Muted:        request.Muted,
		ChannelIds:   request.ChannelIds,
		Enabled:      request.Enabled,
		SilenceUntil: 0,
		Base:         models.Base{Remark: request.Remark, CreateBy: userId, UpdateBy: userId},
	}
}

// EventController 告警事件查询与处理
type EventController struct {
	controllers.BaseController
}

// PageHandler 分页查询告警事件
func (c *EventController) PageHandler(ctx *gin.Context) {
	request := api.EventPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	pageNum, pageSize := normalizePage(request.PageNum, request.PageSize)

	service := monitorService.EventService{}
	result, err := service.List(pageNum, pageSize, monitorService.EventFilter{
		RuleId:     request.RuleId,
		Severity:   request.Severity,
		Status:     request.Status,
		InstanceIp: request.InstanceIp,
		StartAt:    request.StartAt,
		EndAt:      request.EndAt,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// DetailHandler 事件详情与通知历史
func (c *EventController) DetailHandler(ctx *gin.Context) {
	request := api.EventIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少事件ID")
		return
	}
	service := monitorService.EventService{}
	event, logs, err := service.Detail(request.Id)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, gin.H{"event": event, "notifyLogs": logs})
}

// AckHandler 人工处理告警事件
func (c *EventController) AckHandler(ctx *gin.Context) {
	request := api.EventAckRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少事件ID或处理动作")
		return
	}
	service := monitorService.EventService{}
	if err := service.Ack(request.Id, request.Action, request.Remark, c.GetUserName(ctx)); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除已恢复的事件记录
func (c *EventController) DeleteHandler(ctx *gin.Context) {
	request := api.EventIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少事件ID")
		return
	}
	service := monitorService.EventService{}
	if err := service.Delete(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ChannelController 告警通知渠道管理
type ChannelController struct {
	controllers.BaseController
}

// PageHandler 分页查询通知渠道
func (c *ChannelController) PageHandler(ctx *gin.Context) {
	request := api.ChannelPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	pageNum, pageSize := normalizePage(request.PageNum, request.PageSize)

	service := monitorService.ChannelService{}
	result, err := service.List(pageNum, pageSize, monitorService.ChannelFilter{
		Name:    request.Name,
		Type:    request.Type,
		Enabled: request.Enabled,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// OptionsHandler 启用中的渠道下拉列表
func (c *ChannelController) OptionsHandler(ctx *gin.Context) {
	service := monitorService.ChannelService{}
	channels, err := service.Options()
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, channels)
}

// AddHandler 新增通知渠道
func (c *ChannelController) AddHandler(ctx *gin.Context) {
	request := api.ChannelRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：渠道名称与通知地址不能为空")
		return
	}
	userId := c.GetUserId(ctx)
	channel := buildChannel(&request, userId)

	service := monitorService.ChannelService{}
	if err := service.Add(channel); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, gin.H{"id": channel.ID})
}

// EditHandler 编辑通知渠道
func (c *ChannelController) EditHandler(ctx *gin.Context) {
	request := api.ChannelRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：渠道名称与通知地址不能为空")
		return
	}
	if request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少渠道ID")
		return
	}
	userId := c.GetUserId(ctx)
	channel := buildChannel(&request, userId)
	channel.ID = request.Id

	service := monitorService.ChannelService{}
	if err := service.Edit(channel); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除通知渠道
func (c *ChannelController) DeleteHandler(ctx *gin.Context) {
	request := api.ChannelIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少渠道ID")
		return
	}
	service := monitorService.ChannelService{}
	if err := service.Delete(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ToggleHandler 启停通知渠道
func (c *ChannelController) ToggleHandler(ctx *gin.Context) {
	request := api.ChannelIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少渠道ID")
		return
	}
	service := monitorService.ChannelService{}
	if err := service.Toggle(request.Id, request.Enabled); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// TestHandler 向渠道发送测试通知
func (c *ChannelController) TestHandler(ctx *gin.Context) {
	request := api.ChannelIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少渠道ID")
		return
	}
	queryCtx, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()

	service := monitorService.ChannelService{}
	result, err := service.Test(queryCtx, request.Id)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, result)
}

func buildChannel(request *api.ChannelRequest, userId string) *models.OpsMonitorNotifyChannel {
	channelType := request.Type
	if channelType == 0 {
		channelType = models.ChannelTypeWebhook
	}
	return &models.OpsMonitorNotifyChannel{
		Name:    request.Name,
		Type:    channelType,
		Url:     request.Url,
		Secret:  request.Secret,
		Headers: request.Headers,
		Enabled: request.Enabled,
		Base:    models.Base{Remark: request.Remark, CreateBy: userId, UpdateBy: userId},
	}
}

// MetricsController 监控指标查询
type MetricsController struct {
	controllers.BaseController
}

// StatusHandler 数据源与监控配置就绪情况
func (c *MetricsController) StatusHandler(ctx *gin.Context) {
	c.Success(ctx, monitorService.NewMetricService().Status(ctx))
}

// OverviewHandler 告警规模统计
func (c *MetricsController) OverviewHandler(ctx *gin.Context) {
	counts, err := monitorService.OverviewCount()
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, counts)
}

// HostsHandler 已上报监控数据的主机列表
func (c *MetricsController) HostsHandler(ctx *gin.Context) {
	service := monitorService.NewMetricService()
	hosts, err := service.Hosts(ctx)
	if err != nil {
		c.Failure(ctx, http.StatusServiceUnavailable, err.Error())
		return
	}
	c.Success(ctx, hosts)
}

// InstantHandler 主机当前指标快照
func (c *MetricsController) InstantHandler(ctx *gin.Context) {
	request := api.MetricsHostRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少主机地址")
		return
	}
	service := monitorService.NewMetricService()
	snapshot, err := service.Snapshot(ctx, request.Instance)
	if err != nil {
		c.Failure(ctx, http.StatusServiceUnavailable, err.Error())
		return
	}
	c.Success(ctx, snapshot)
}

// RangeHandler 指标趋势数据
func (c *MetricsController) RangeHandler(ctx *gin.Context) {
	request := api.MetricsRangeRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少监控指标")
		return
	}
	query := monitorService.RangeQuery{
		Metric:     request.Metric,
		Instance:   request.Instance,
		Mountpoint: request.Mountpoint,
		Device:     request.Device,
		StepSec:    request.StepSec,
	}
	if request.StartAt > 0 {
		query.Start = time.Unix(request.StartAt, 0)
	}
	if request.EndAt > 0 {
		query.End = time.Unix(request.EndAt, 0)
	}

	service := monitorService.NewMetricService()
	series, err := service.Range(ctx, query)
	if err != nil {
		c.Failure(ctx, http.StatusServiceUnavailable, err.Error())
		return
	}
	c.Success(ctx, series)
}
