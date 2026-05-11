package instance

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/instance"
	"net/http"
	"strconv"
)

type ScriptController struct {
	controllers.BaseController
}

func (c *ScriptController) ListHandler(ctx *gin.Context) {
	request := api.PageScriptRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil && err.Error() != "EOF" {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if request.PageNum <= 0 {
		request.PageNum = 1
	}
	if request.PageSize <= 0 {
		request.PageSize = 10
	}
	service := instance.ScriptService{}
	result, err := service.List(request.PageNum, request.PageSize, request.Name)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

func (c *ScriptController) AddHandler(ctx *gin.Context) {
	request := api.AddScriptRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userId := c.GetUserId(ctx)
	script := &models.OpsScript{
		Name:    request.Name,
		Content: request.Content,
		Type:    request.Type,
		Remark:  request.Remark,
	}
	script.CreateBy = userId
	script.UpdateBy = userId
	service := instance.ScriptService{}
	if err := service.Add(script); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

func (c *ScriptController) EditHandler(ctx *gin.Context) {
	request := api.UpdateScriptRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userId := c.GetUserId(ctx)
	script := &models.OpsScript{
		Name:    request.Name,
		Content: request.Content,
		Type:    request.Type,
		Remark:  request.Remark,
	}
	script.ID = request.Id
	script.UpdateBy = userId
	service := instance.ScriptService{}
	if err := service.Edit(script); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

func (c *ScriptController) DeleteHandler(ctx *gin.Context) {
	requestId := ctx.Param("id")
	id := 0
	if _, _ = fmt.Sscanf(requestId, "%d", &id); id == 0 {
		c.Failure(ctx, http.StatusBadRequest, "无效的ID")
		return
	}
	service := instance.ScriptService{}
	if err := service.Delete(id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

func (c *ScriptController) DetailHandler(ctx *gin.Context) {
	requestId := ctx.Param("id")
	id, _ := strconv.Atoi(requestId)
	service := instance.ScriptService{}
	result, err := service.GetByID(id)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, result)
}
