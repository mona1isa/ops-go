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

// TargetController 监控采集目标管理
type TargetController struct {
	controllers.BaseController
}

// PageHandler 分页查询采集目标
func (c *TargetController) PageHandler(ctx *gin.Context) {
	request := api.TargetPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	pageNum, pageSize := normalizePage(request.PageNum, request.PageSize)

	service := monitorService.TargetService{}
	result, err := service.List(pageNum, pageSize, monitorService.TargetFilter{
		Name:       request.Name,
		InstanceIp: request.InstanceIp,
		LastStatus: request.LastStatus,
		Enabled:    request.Enabled,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AddHandler 新增采集目标
func (c *TargetController) AddHandler(ctx *gin.Context) {
	request := api.TargetRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：目标主机不能为空")
		return
	}
	userId := c.GetUserId(ctx)
	target := &models.OpsMonitorTarget{
		InstanceId:  request.InstanceId,
		KeyId:       request.KeyId,
		IntervalSec: request.IntervalSec,
		TimeoutSec:  request.TimeoutSec,
		Enabled:     request.Enabled,
		Base:        models.Base{Remark: request.Remark, CreateBy: userId, UpdateBy: userId},
	}

	service := monitorService.TargetService{}
	if err := service.Add(target); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// EditHandler 编辑采集目标
func (c *TargetController) EditHandler(ctx *gin.Context) {
	request := api.TargetRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少采集目标ID")
		return
	}
	userId := c.GetUserId(ctx)
	target := &models.OpsMonitorTarget{
		InstanceId:  request.InstanceId,
		KeyId:       request.KeyId,
		IntervalSec: request.IntervalSec,
		TimeoutSec:  request.TimeoutSec,
		Enabled:     request.Enabled,
		Base:        models.Base{ID: request.Id, Remark: request.Remark, UpdateBy: userId},
	}

	service := monitorService.TargetService{}
	if err := service.Edit(target); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除采集目标
func (c *TargetController) DeleteHandler(ctx *gin.Context) {
	request := api.TargetIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少采集目标ID")
		return
	}
	service := monitorService.TargetService{}
	if err := service.Delete(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ToggleHandler 启停采集目标
func (c *TargetController) ToggleHandler(ctx *gin.Context) {
	request := api.TargetToggleRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少采集目标ID或启用状态")
		return
	}
	service := monitorService.TargetService{}
	if err := service.Toggle(request.Id, request.Enabled); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// SyncHandler 从主机列表批量补齐采集目标
func (c *TargetController) SyncHandler(ctx *gin.Context) {
	service := monitorService.TargetService{}
	created, err := service.Sync(c.GetUserId(ctx))
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, gin.H{"created": created})
}

// CollectHandler 对单个采集目标手动执行一次采集，即时返回成败与原因
func (c *TargetController) CollectHandler(ctx *gin.Context) {
	request := api.TargetIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少采集目标ID")
		return
	}
	queryCtx, cancel := context.WithTimeout(ctx.Request.Context(), 40*time.Second)
	defer cancel()

	result, err := monitorService.NewCollectorService().CollectTargetNow(queryCtx, request.Id)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, result)
}

// InstanceKeysHandler 查询指定主机已绑定的凭证，供采集目标表单凭证下拉联动
func (c *TargetController) InstanceKeysHandler(ctx *gin.Context) {
	request := api.TargetInstanceKeysRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少主机ID")
		return
	}
	service := monitorService.TargetService{}
	keys, err := service.InstanceKeys(request.InstanceId)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, keys)
}

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
