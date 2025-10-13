package instance

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/services/instance"
	"net/http"
	"strconv"
)

type GroupController struct {
	controllers.BaseController
}

// AddGroupHandler 添加分组
func (c *GroupController) AddGroupHandler(ctx *gin.Context) {
	request := api.AddGroupRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.GroupService{}
	if err := service.AddGroup(request); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// EditGroupHandler 编辑分组
func (c *GroupController) EditGroupHandler(ctx *gin.Context) {
	request := api.UpdateGroupRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.GroupService{}
	if err := service.EditGroup(request); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// PageGroupHandler 分页查询分组
func (c *GroupController) GroupTreeHandler(ctx *gin.Context) {
	service := instance.GroupService{}
	data, err := service.ListGroup()
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	c.Success(ctx, data)
}

// DeleteGroupHandler 删除分组
func (c *GroupController) DeleteGroupHandler(ctx *gin.Context) {
	requestId := ctx.Param("id")
	id, _ := strconv.Atoi(requestId)
	service := instance.GroupService{}
	if err := service.DeleteGroup(id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// GroupInstanceHandler 添加分组实例
func (c *GroupController) GroupInstanceHandler(ctx *gin.Context) {
	request := api.GroupInstanceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := instance.GroupService{}
	if err := service.GroupInstanceOps(request); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// PageGroupInstanceHandler 获取分组实例列表
func (c *GroupController) PageGroupInstanceHandler(ctx *gin.Context) {
	request := api.PageGroupInstanceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := instance.GroupService{}
	instances, err := service.PageGroupInstance(request)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, instances)
}

// AvailableInstanceHandler 查询可添加到分组的实例
func (c *GroupController) AvailableInstanceHandler(ctx *gin.Context) {
	request := api.PageGroupInstanceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := instance.GroupService{}
	instances, err := service.AvailableInstance(request)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.Success(ctx, instances)
}
