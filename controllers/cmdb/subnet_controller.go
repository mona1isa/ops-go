package cmdb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	api "github.com/zhany/ops-go/controllers/cmdb/api"
	"github.com/zhany/ops-go/models"
	cmdbService "github.com/zhany/ops-go/services/cmdb"
)

// SubnetController IPAM 网段管理
type SubnetController struct {
	controllers.BaseController
}

// PageHandler 网段分页查询
func (c *SubnetController) PageHandler(ctx *gin.Context) {
	request := api.SubnetPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	pageNum, pageSize := normalizeCmdbPage(request.PageNum, request.PageSize)
	service := cmdbService.SubnetService{}
	result, err := service.List(pageNum, pageSize, request.Keyword)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AddHandler 新增网段
func (c *SubnetController) AddHandler(ctx *gin.Context) {
	request := api.SubnetRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：网段名称与 CIDR 不能为空")
		return
	}
	userId := c.GetUserId(ctx)
	subnet := &models.OpsCmdbSubnet{
		Name:    request.Name,
		Cidr:    request.Cidr,
		Gateway: request.Gateway,
		Vlan:    request.Vlan,
		Base:    models.Base{Remark: request.Remark, CreateBy: userId, UpdateBy: userId},
	}
	service := cmdbService.SubnetService{}
	if err := service.Add(subnet); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, gin.H{"id": subnet.ID})
}

// EditHandler 编辑网段
func (c *SubnetController) EditHandler(ctx *gin.Context) {
	request := api.SubnetRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Id <= 0 {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少网段ID")
		return
	}
	userId := c.GetUserId(ctx)
	subnet := &models.OpsCmdbSubnet{
		Name:    request.Name,
		Cidr:    request.Cidr,
		Gateway: request.Gateway,
		Vlan:    request.Vlan,
		Base:    models.Base{ID: request.Id, Remark: request.Remark, UpdateBy: userId},
	}
	service := cmdbService.SubnetService{}
	if err := service.Edit(subnet); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除网段
func (c *SubnetController) DeleteHandler(ctx *gin.Context) {
	request := api.SubnetIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少网段ID")
		return
	}
	service := cmdbService.SubnetService{}
	if err := service.Delete(request.Id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// UsageHandler 网段 IP 占用情况
func (c *SubnetController) UsageHandler(ctx *gin.Context) {
	request := api.SubnetIdRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法：缺少网段ID")
		return
	}
	service := cmdbService.SubnetService{}
	usage, err := service.Usage(request.Id)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, usage)
}
