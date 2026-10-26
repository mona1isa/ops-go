package cmdb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	api "github.com/zhany/ops-go/controllers/cmdb/api"
	monitorApi "github.com/zhany/ops-go/controllers/monitor/api"
	"github.com/zhany/ops-go/models"
	cmdbService "github.com/zhany/ops-go/services/cmdb"
	monitorService "github.com/zhany/ops-go/services/monitor"
)

// CertAlertController 证书/域名到期告警：规则管理与到期概览
type CertAlertController struct {
	controllers.BaseController
}

// RulePageHandler 规则分页
func (c *CertAlertController) RulePageHandler(ctx *gin.Context) {
	request := api.CertRulePageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	pageNum, pageSize := normalizeCmdbPage(request.PageNum, request.PageSize)

	service := cmdbService.CertAlertService{}
	result, err := service.ListRules(pageNum, pageSize, cmdbService.CertRulePageRequest{
		Name: request.Name, Enabled: request.Enabled, Muted: request.Muted,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// RuleAddHandler 新增规则
func (c *CertAlertController) RuleAddHandler(ctx *gin.Context) {
	request := api.CertRuleRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	rule := &models.OpsCertAlertRule{
		Name: request.Name, ModelTypes: request.ModelTypes, WarningDays: request.WarningDays,
		Severity: request.Severity, CooldownSec: request.CooldownSec, Muted: request.Muted,
		ChannelIds: request.ChannelIds, Enabled: request.Enabled,
		Base: models.Base{Remark: request.Remark, CreateBy: c.GetUserId(ctx), UpdateBy: c.GetUserId(ctx)},
	}
	service := cmdbService.CertAlertService{}
	if err := service.AddRule(rule); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.Success(ctx, gin.H{"id": rule.ID})
}

// RuleEditHandler 编辑规则
func (c *CertAlertController) RuleEditHandler(ctx *gin.Context) {
	request := api.CertRuleRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	if request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "缺少规则ID")
		return
	}
	userId := c.GetUserId(ctx)
	rule := &models.OpsCertAlertRule{
		Base:        models.Base{ID: request.Id, Remark: request.Remark, UpdateBy: userId},
		Name:        request.Name, ModelTypes: request.ModelTypes, WarningDays: request.WarningDays,
		Severity: request.Severity, CooldownSec: request.CooldownSec, Muted: request.Muted,
		ChannelIds: request.ChannelIds, Enabled: request.Enabled,
	}
	service := cmdbService.CertAlertService{}
	if err := service.EditRule(rule); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// RuleDeleteHandler 删除规则
func (c *CertAlertController) RuleDeleteHandler(ctx *gin.Context) {
	request := api.CertRuleIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少规则ID")
		return
	}
	service := cmdbService.CertAlertService{}
	if err := service.DeleteRule(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// RuleToggleHandler 启停规则
func (c *CertAlertController) RuleToggleHandler(ctx *gin.Context) {
	request := api.CertRuleToggleRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	service := cmdbService.CertAlertService{}
	if err := service.ToggleRule(request.Id, request.Enabled); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// RuleSilenceHandler 静默规则，minute<=0 表示立即解除
func (c *CertAlertController) RuleSilenceHandler(ctx *gin.Context) {
	request := api.CertRuleSilenceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	service := cmdbService.CertAlertService{}
	if err := service.SilenceRule(request.Id, request.Minute); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ExpiringHandler 到期概览：预警窗口内（含已过期）的资产，按到期时间升序
func (c *CertAlertController) ExpiringHandler(ctx *gin.Context) {
	request := api.CertExpiringRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		request = api.CertExpiringRequest{}
	}
	service := cmdbService.CertAlertService{}
	rows, err := service.ExpiringAssets(request.ModelTypes, request.WarningDays)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, rows)
}

// EventsHandler 证书到期告警事件列表，复用监控告警事件表，按来源 cert 过滤
func (c *CertAlertController) EventsHandler(ctx *gin.Context) {
	request := monitorApi.EventPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	pageNum, pageSize := normalizeCmdbPage(request.PageNum, request.PageSize)

	service := monitorService.EventService{}
	result, err := service.List(pageNum, pageSize, monitorService.EventFilter{
		RuleId:   request.RuleId,
		Severity: request.Severity,
		Status:   request.Status,
		Source:   models.AlertSourceCert,
		StartAt:  request.StartAt,
		EndAt:    request.EndAt,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}
