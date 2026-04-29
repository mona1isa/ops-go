package instance

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/instance"
)

type DangerousCommandController struct {
	controllers.BaseController
}

// ListHandler 分页查询高危指令规则
func (c *DangerousCommandController) ListHandler(ctx *gin.Context) {
	request := api.DangerousCommandPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if request.PageNum == 0 {
		request.PageNum = 1
	}
	if request.PageSize == 0 {
		request.PageSize = 10
	}

	service := instance.DangerousCommandService{}
	result, err := service.List(request.PageNum, request.PageSize, request.Name)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AddHandler 新增高危指令规则
func (c *DangerousCommandController) AddHandler(ctx *gin.Context) {
	request := api.DangerousCommandRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := c.GetUserId(ctx)
	command := &models.OpsDangerousCommand{
		Name:        request.Name,
		Pattern:     request.Pattern,
		MatchType:   request.MatchType,
		Description: request.Description,
		IsEnabled:   request.IsEnabled,
		IsBuiltin:   0,
	}
	command.CreateBy = userId
	command.UpdateBy = userId

	service := instance.DangerousCommandService{}
	if err := service.Add(command); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// EditHandler 编辑高危指令规则
func (c *DangerousCommandController) EditHandler(ctx *gin.Context) {
	request := api.DangerousCommandRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := c.GetUserId(ctx)
	command := &models.OpsDangerousCommand{
		Name:        request.Name,
		Pattern:     request.Pattern,
		MatchType:   request.MatchType,
		Description: request.Description,
		IsEnabled:   request.IsEnabled,
	}
	command.ID = request.Id
	command.UpdateBy = userId

	service := instance.DangerousCommandService{}
	if err := service.Edit(command); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// DeleteHandler 删除高危指令规则
func (c *DangerousCommandController) DeleteHandler(ctx *gin.Context) {
	requestId := ctx.Param("id")
	id, _ := strconv.Atoi(requestId)
	if id == 0 {
		c.Failure(ctx, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	service := instance.DangerousCommandService{}
	if err := service.Delete(id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// ChangeStatusHandler 切换规则启用状态
func (c *DangerousCommandController) ChangeStatusHandler(ctx *gin.Context) {
	request := api.ChangeDangerousCommandStatusRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.DangerousCommandService{}
	if err := service.ToggleStatus(request.Id, request.IsEnabled); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	c.JustSuccess(ctx)
}
