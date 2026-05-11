package instance

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/instance"
)

type TaskExecutionController struct {
	controllers.BaseController
}

// QuickExecuteHandler 快速执行
func (c *TaskExecutionController) QuickExecuteHandler(ctx *gin.Context) {
	request := api.QuickExecuteRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := instance.ParseUserId(c.GetUserId(ctx))
	userName := c.GetUserName(ctx)
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = 300
	}
	name := request.Name
	if name == "" {
		name = "快速执行"
	}

	service := instance.TaskExecutionService{}
	execution, err := service.CreateExecution(name, request.Type, 0, userId, userName, request.InstanceIds, request.KeyId, timeout, request.Content, request.ScriptLang, request.SrcPath, request.DestPath)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// 异步执行
	go instance.RunExecution(execution.ID)

	c.Success(ctx, gin.H{"executionId": execution.ID, "executionNo": execution.ExecutionNo})
}

// TemplateExecuteHandler 按模板执行
func (c *TaskExecutionController) TemplateExecuteHandler(ctx *gin.Context) {
	request := api.TemplateExecuteRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := instance.ParseUserId(c.GetUserId(ctx))
	userName := c.GetUserName(ctx)

	// 查询模板获取超时时间
	templateService := instance.TaskTemplateService{}
	tpl, err := templateService.GetByID(request.TemplateId)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = tpl.Timeout
	}

	service := instance.TaskExecutionService{}
	execution, err := service.CreateExecution(tpl.Name, models.ExecTypeTemplate, request.TemplateId, userId, userName, request.InstanceIds, request.KeyId, timeout, "", "", "", "")
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// 异步执行
	go instance.RunExecution(execution.ID)

	c.Success(ctx, gin.H{"executionId": execution.ID, "executionNo": execution.ExecutionNo})
}

// PipelineExecuteHandler 执行编排
func (c *TaskExecutionController) PipelineExecuteHandler(ctx *gin.Context) {
	request := api.PipelineExecuteRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := instance.ParseUserId(c.GetUserId(ctx))
	userName := c.GetUserName(ctx)

	// 查询编排
	pipelineService := instance.TaskPipelineService{}
	pipeline, err := pipelineService.GetByID(request.PipelineId)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.TaskExecutionService{}
	execution, err := service.CreateExecution(pipeline.Name, models.ExecTypePipeline, request.PipelineId, userId, userName, request.InstanceIds, request.KeyId, 0, "", "", "", "")
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// 异步执行编排
	go instance.RunPipelineExecution(execution.ID)

	c.Success(ctx, gin.H{"executionId": execution.ID, "executionNo": execution.ExecutionNo})
}

// CancelHandler 取消执行
func (c *TaskExecutionController) CancelHandler(ctx *gin.Context) {
	request := api.CancelExecutionRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := instance.ParseUserId(c.GetUserId(ctx))
	service := instance.TaskExecutionService{}
	if err := service.CancelExecution(request.ExecutionId, userId); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// ListHandler 分页查询执行记录
func (c *TaskExecutionController) ListHandler(ctx *gin.Context) {
	request := api.PageExecutionRequest{}
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

	service := instance.TaskExecutionService{}
	result, err := service.List(request.PageNum, request.PageSize, request.Status, request.Type, request.StartAt, request.EndAt)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// DetailHandler 执行详情
func (c *TaskExecutionController) DetailHandler(ctx *gin.Context) {
	request := api.ExecutionDetailRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.TaskExecutionService{}
	result, err := service.GetByID(request.ExecutionId)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// 同时获取主机结果
	hosts, _ := service.GetExecutionHosts(request.ExecutionId)

	// 编排执行额外返回步骤及每步的主机结果
	if result.Type == models.ExecTypePipeline {
		steps, _ := service.GetStepExecutions(request.ExecutionId)
		var stepDetails []gin.H
		for _, step := range steps {
			stepHosts, _ := service.GetExecutionHostsByStepExecId(request.ExecutionId, step.ID)
			stepDetails = append(stepDetails, gin.H{
				"step":  step,
				"hosts": stepHosts,
			})
		}
		c.Success(ctx, gin.H{
			"execution": result,
			"hosts":     hosts,
			"steps":     stepDetails,
		})
		return
	}

	c.Success(ctx, gin.H{
		"execution": result,
		"hosts":     hosts,
	})
}

// HostResultHandler 单台主机执行结果
func (c *TaskExecutionController) HostResultHandler(ctx *gin.Context) {
	request := api.HostResultRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.TaskExecutionService{}
	result, err := service.GetHostResult(request.ExecutionId, request.InstanceId)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, result)
}
