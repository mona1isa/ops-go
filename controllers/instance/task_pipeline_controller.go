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

type TaskPipelineController struct {
	controllers.BaseController
}

// ListHandler 分页查询编排
func (c *TaskPipelineController) ListHandler(ctx *gin.Context) {
	request := api.PagePipelineRequest{}
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

	service := instance.TaskPipelineService{}
	result, err := service.List(request.PageNum, request.PageSize, request.Name)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AddHandler 新增编排
func (c *TaskPipelineController) AddHandler(ctx *gin.Context) {
	request := api.AddPipelineRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := c.GetUserId(ctx)
	pipeline := &models.OpsTaskPipeline{
		Name:        request.Name,
		Description: request.Description,
	}
	pipeline.CreateBy = userId
	pipeline.UpdateBy = userId

	// 构建步骤
	steps := make([]models.OpsPipelineStep, 0, len(request.Steps))
	for _, s := range request.Steps {
		steps = append(steps, models.OpsPipelineStep{
			StepName:     s.StepName,
			TemplateId:   s.TemplateId,
			StepOrder:    s.StepOrder,
			ParentStepId: s.ParentStepId,
			OnFailure:   s.OnFailure,
			RetryCount:   s.RetryCount,
		})
	}
	pipeline.Steps = steps

	service := instance.TaskPipelineService{}
	if err := service.Add(pipeline); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// EditHandler 编辑编排
func (c *TaskPipelineController) EditHandler(ctx *gin.Context) {
	request := api.UpdatePipelineRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := c.GetUserId(ctx)
	pipeline := &models.OpsTaskPipeline{
		Name:        request.Name,
		Description: request.Description,
	}
	pipeline.ID = request.Id
	pipeline.UpdateBy = userId

	steps := make([]models.OpsPipelineStep, 0, len(request.Steps))
	for _, s := range request.Steps {
		steps = append(steps, models.OpsPipelineStep{
			StepName:     s.StepName,
			TemplateId:   s.TemplateId,
			StepOrder:    s.StepOrder,
			ParentStepId: s.ParentStepId,
			OnFailure:   s.OnFailure,
			RetryCount:   s.RetryCount,
		})
	}
	pipeline.Steps = steps

	service := instance.TaskPipelineService{}
	if err := service.Edit(pipeline); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DeleteHandler 删除编排
func (c *TaskPipelineController) DeleteHandler(ctx *gin.Context) {
	requestId := ctx.Param("id")
	id := 0
	if _, _ = fmt.Sscanf(requestId, "%d", &id); id == 0 {
		c.Failure(ctx, http.StatusBadRequest, "编排ID不能为空")
		return
	}

	service := instance.TaskPipelineService{}
	if err := service.Delete(id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// DetailHandler 查询编排详情
func (c *TaskPipelineController) DetailHandler(ctx *gin.Context) {
	request := struct {
		Id int `json:"id" binding:"required"`
	}{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.TaskPipelineService{}
	result, err := service.GetByID(request.Id)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, result)
}
