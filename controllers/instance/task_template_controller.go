package instance

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/instance"
)

type TaskTemplateController struct {
	controllers.BaseController
}

// ListHandler 分页查询任务模板
func (c *TaskTemplateController) ListHandler(ctx *gin.Context) {
	request := api.PageTaskTemplateRequest{}
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

	service := instance.TaskTemplateService{}
	result, err := service.List(request.PageNum, request.PageSize, request.Name, request.Type)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AddHandler 新增任务模板
func (c *TaskTemplateController) AddHandler(ctx *gin.Context) {
	request := api.AddTaskTemplateRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := c.GetUserId(ctx)
	template := &models.OpsTaskTemplate{
		Name:        request.Name,
		Type:        request.Type,
		Content:     request.Content,
		ScriptLang:  request.ScriptLang,
		SrcPath:     request.SrcPath,
		DestPath:    request.DestPath,
		Timeout:     request.Timeout,
		KeyId:       request.KeyId,
		Description: request.Description,
	}
	template.CreateBy = userId
	template.UpdateBy = userId

	service := instance.TaskTemplateService{}
	if err := service.Add(template); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// EditHandler 编辑任务模板
func (c *TaskTemplateController) EditHandler(ctx *gin.Context) {
	request := api.UpdateTaskTemplateRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := c.GetUserId(ctx)
	template := &models.OpsTaskTemplate{
		Name:        request.Name,
		Type:        request.Type,
		Content:     request.Content,
		ScriptLang:  request.ScriptLang,
		SrcPath:     request.SrcPath,
		DestPath:    request.DestPath,
		Timeout:     request.Timeout,
		KeyId:       request.KeyId,
		Description: request.Description,
	}
	template.ID = request.Id
	template.UpdateBy = userId

	service := instance.TaskTemplateService{}
	if err := service.Edit(template); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除任务模板
func (c *TaskTemplateController) DeleteHandler(ctx *gin.Context) {
	requestId := ctx.Param("id")
	id := 0
	if _, _ = fmt.Sscanf(requestId, "%d", &id); id == 0 {
		c.Failure(ctx, http.StatusBadRequest, "模板ID不能为空")
		return
	}

	service := instance.TaskTemplateService{}
	if err := service.Delete(id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DetailHandler 查询模板详情
func (c *TaskTemplateController) DetailHandler(ctx *gin.Context) {
	request := api.UpdateTaskTemplateRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.TaskTemplateService{}
	result, err := service.GetByID(request.Id)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, result)
}
