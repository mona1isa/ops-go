package cmdb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	api "github.com/zhany/ops-go/controllers/cmdb/api"
	"github.com/zhany/ops-go/models"
	cmdbService "github.com/zhany/ops-go/services/cmdb"
)

// AssetController 资产台账管理
type AssetController struct {
	controllers.BaseController
}

// PageHandler 分页查询资产
func (c *AssetController) PageHandler(ctx *gin.Context) {
	request := api.AssetPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	pageNum, pageSize := normalizeCmdbPage(request.PageNum, request.PageSize)

	service := cmdbService.AssetService{}
	result, err := service.List(pageNum, pageSize, cmdbService.AssetFilter{
		ModelType:    request.ModelType,
		Status:       request.Status,
		TagId:        request.TagId,
		Keyword:      request.Keyword,
		ExpireWithin: request.ExpireWithin,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AddHandler 新增资产
func (c *AssetController) AddHandler(ctx *gin.Context) {
	request := api.AssetRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：资产模型类型与名称不能为空")
		return
	}
	userId := c.GetUserId(ctx)
	asset := &models.OpsCmdbAsset{
		ModelType: request.ModelType,
		Name:      request.Name,
		Sn:        request.Sn,
		Ip:        request.Ip,
		Status:    request.Status,
		Owner:     request.Owner,
		Provider:  request.Provider,
		Location:  request.Location,
		ExpiredAt: request.ExpiredAt,
		Attrs:     request.Attrs,
		Base:      models.Base{Remark: request.Remark, CreateBy: userId, UpdateBy: userId},
	}
	service := cmdbService.AssetService{}
	if err := service.Add(asset); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, gin.H{"id": asset.ID})
}

// EditHandler 编辑资产
func (c *AssetController) EditHandler(ctx *gin.Context) {
	request := api.AssetRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少资产ID")
		return
	}
	userId := c.GetUserId(ctx)
	asset := &models.OpsCmdbAsset{
		Name:      request.Name,
		Sn:        request.Sn,
		Ip:        request.Ip,
		Owner:     request.Owner,
		Provider:  request.Provider,
		Location:  request.Location,
		ExpiredAt: request.ExpiredAt,
		Attrs:     request.Attrs,
		Base:      models.Base{ID: request.Id, Remark: request.Remark, UpdateBy: userId},
	}
	service := cmdbService.AssetService{}
	if err := service.Edit(asset); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DetailHandler 资产详情
func (c *AssetController) DetailHandler(ctx *gin.Context) {
	request := api.AssetIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少资产ID")
		return
	}
	service := cmdbService.AssetService{}
	detail, err := service.Detail(request.Id)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, detail)
}

// DeleteHandler 删除资产
func (c *AssetController) DeleteHandler(ctx *gin.Context) {
	request := api.AssetIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少资产ID")
		return
	}
	service := cmdbService.AssetService{}
	if err := service.Delete(request.Id, c.GetUserId(ctx)); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// TransitionHandler 资产生命周期状态流转
func (c *AssetController) TransitionHandler(ctx *gin.Context) {
	request := api.AssetTransitionRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少资产ID或目标状态")
		return
	}
	service := cmdbService.AssetService{}
	if err := service.Transition(request.Id, request.Status, request.Remark, c.GetUserId(ctx)); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ImportHandler 从已纳管主机批量导入服务器资产
func (c *AssetController) ImportHandler(ctx *gin.Context) {
	service := cmdbService.AssetService{}
	created, err := service.ImportFromInstances(c.GetUserId(ctx))
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, gin.H{"created": created})
}

// OptionsHandler 资产下拉选项（关联目标选择器）
func (c *AssetController) OptionsHandler(ctx *gin.Context) {
	request := api.AssetOptionsRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		request.Keyword = ""
	}
	service := cmdbService.AssetService{}
	options, err := service.AssetOptions(request.Keyword)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, options)
}

// ChangeLogHandler 资产变更记录分页
func (c *AssetController) ChangeLogHandler(ctx *gin.Context) {
	request := api.ChangeLogRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少资产ID")
		return
	}
	pageNum, pageSize := normalizeCmdbPage(request.PageNum, request.PageSize)
	service := cmdbService.ChangeLogService{}
	result, err := service.List(request.AssetId, pageNum, pageSize)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// normalizeCmdbPage 统一分页默认值与上限
func normalizeCmdbPage(pageNum, pageSize int) (int, int) {
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
