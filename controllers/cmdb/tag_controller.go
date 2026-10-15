package cmdb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	api "github.com/zhany/ops-go/controllers/cmdb/api"
	"github.com/zhany/ops-go/models"
	cmdbService "github.com/zhany/ops-go/services/cmdb"
)

// TagController 资产标签管理
type TagController struct {
	controllers.BaseController
}

// ListHandler 全量标签列表
func (c *TagController) ListHandler(ctx *gin.Context) {
	service := cmdbService.TagService{}
	tags, err := service.List()
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, tags)
}

// AddHandler 新增标签
func (c *TagController) AddHandler(ctx *gin.Context) {
	request := api.TagRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：标签名称不能为空")
		return
	}
	userId := c.GetUserId(ctx)
	tag := &models.OpsCmdbTag{
		Name:  request.Name,
		Color: request.Color,
		Base:  models.Base{CreateBy: userId, UpdateBy: userId},
	}
	service := cmdbService.TagService{}
	if err := service.Add(tag); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, gin.H{"id": tag.ID})
}

// EditHandler 编辑标签
func (c *TagController) EditHandler(ctx *gin.Context) {
	request := api.TagRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少标签ID")
		return
	}
	userId := c.GetUserId(ctx)
	tag := &models.OpsCmdbTag{
		Name:  request.Name,
		Color: request.Color,
		Base:  models.Base{ID: request.Id, UpdateBy: userId},
	}
	service := cmdbService.TagService{}
	if err := service.Edit(tag); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除标签
func (c *TagController) DeleteHandler(ctx *gin.Context) {
	request := api.TagIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少标签ID")
		return
	}
	service := cmdbService.TagService{}
	if err := service.Delete(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// SetHandler 全量设置资产标签
func (c *TagController) SetHandler(ctx *gin.Context) {
	request := api.TagSetRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少资产ID")
		return
	}
	service := cmdbService.TagService{}
	if err := service.SetAssetTags(request.AssetId, request.TagIds, c.GetUserId(ctx)); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}
