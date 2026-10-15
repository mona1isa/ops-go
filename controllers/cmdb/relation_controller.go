package cmdb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	api "github.com/zhany/ops-go/controllers/cmdb/api"
	cmdbService "github.com/zhany/ops-go/services/cmdb"
)

// RelationController 资产关联关系与拓扑
type RelationController struct {
	controllers.BaseController
}

// AddHandler 新增关联关系
func (c *RelationController) AddHandler(ctx *gin.Context) {
	request := api.RelationAddRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少源/目标资产或关系类型")
		return
	}
	service := cmdbService.RelationService{}
	if err := service.Add(request.SourceId, request.TargetId, request.RelationType, c.GetUserId(ctx)); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除关联关系
func (c *RelationController) DeleteHandler(ctx *gin.Context) {
	request := api.RelationIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少关联ID")
		return
	}
	service := cmdbService.RelationService{}
	if err := service.Delete(request.Id, c.GetUserId(ctx)); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ListHandler 查询某资产的关联列表
func (c *RelationController) ListHandler(ctx *gin.Context) {
	request := api.RelationListRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少资产ID")
		return
	}
	service := cmdbService.RelationService{}
	rows, err := service.ListByAsset(request.AssetId)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, rows)
}

// GraphAllHandler 全量拓扑
func (c *RelationController) GraphAllHandler(ctx *gin.Context) {
	request := api.GraphAllRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		request.ModelTypes = nil
	}
	service := cmdbService.RelationService{}
	graph, err := service.GraphAll(request.ModelTypes)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, graph)
}

// GraphHandler 以某资产为中心的邻域拓扑
func (c *RelationController) GraphHandler(ctx *gin.Context) {
	request := api.GraphRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少资产ID")
		return
	}
	service := cmdbService.RelationService{}
	graph, err := service.Graph(request.RootId, request.Depth)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, graph)
}
