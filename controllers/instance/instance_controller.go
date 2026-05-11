package instance

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/services/instance"
	"io"
	"net/http"
	"strconv"
)

type InstanceController struct {
	controllers.BaseController
}

// AddInstanceHandler 添加实例
func (c *InstanceController) AddInstanceHandler(ctx *gin.Context) {
	instanceRequest := api.AddInstanceRequest{}
	if err := ctx.ShouldBindJSON(&instanceRequest); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := c.GetUserId(ctx)
	instanceRequest.CreateBy = userId
	instanceRequest.UpdateBy = userId

	service := instance.InstanceService{}
	err := service.AddInstance(instanceRequest)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// EditInstanceHandler 更新实例信息
func (c *InstanceController) EditInstanceHandler(ctx *gin.Context) {
	updateInstanceRequest := api.UpdateInstanceRequest{}
	if err := ctx.ShouldBindJSON(&updateInstanceRequest); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.InstanceService{}
	err := service.EditInstance(updateInstanceRequest)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// ChangeStatusHandler 修改实例状态
func (c *InstanceController) ChangeStatus(ctx *gin.Context) {
	request := api.ChangeStatusRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.InstanceService{}
	if err := service.ChangeStatus(request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// ListInstanceHandler 查询实例列表（不分页）
func (c *InstanceController) ListInstanceHandler(ctx *gin.Context) {
	request := api.ListInstanceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := instance.InstanceService{}
	info, err := service.ListInstance(request)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.Success(ctx, info)
}

// PageInstanceHandler 分页查询实例
func (c *InstanceController) PageInstanceHandler(ctx *gin.Context) {
	request := api.PageInstanceRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := instance.InstanceService{}
	info, err := service.PageInstance(request)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.Success(ctx, info)
}

// GetInstanceDetailHandler 获取实例详细信息
func (c *InstanceController) GetInstanceDetailHandler(ctx *gin.Context) {
	requestId := ctx.Param("id")
	id, _ := strconv.Atoi(requestId)

	service := instance.InstanceService{}
	info, err := service.GetInstanceDetail(id)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.Success(ctx, info)
}

// DeleteInstanceHandler 删除实例
func (c *InstanceController) DeleteInstanceHandler(ctx *gin.Context) {
	requestId := ctx.Param("id")
	id, _ := strconv.Atoi(requestId)
	service := instance.InstanceService{}
	err := service.DeleteInstance(id)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// KeyBindingHandler 绑定key
func (c *InstanceController) KeyBindingHandler(ctx *gin.Context) {
	request := api.InstanceKeyBindingRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.InstanceService{}
	err := service.KeyBinding(request)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// UnBindingKeyHandler 解绑key
func (c *InstanceController) UnBindingKeyHandler(ctx *gin.Context) {
	request := api.InstanceKeyUnbindingRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := instance.InstanceService{}
	if err := service.UnBindingKey(request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// TestConnectHandler 测试连接
func (c *InstanceController) TestConnectHandler(ctx *gin.Context) {
	request := api.InstanceKeyBindingRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := instance.InstanceService{}
	err := service.TestConnect(request)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.JustSuccess(ctx)
}

// GetMyInstanceHandler  获取我的实例
func (c *InstanceController) GetMyInstanceHandler(ctx *gin.Context) {
	myInstance := new(instance.MyInstance)
	if err := ctx.ShouldBindJSON(&myInstance); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId, _ := strconv.Atoi(c.GetUserId(ctx))
	myInstance.UserId = userId
	myInstance.IsAdmin = c.IsAdminUser(ctx)
	result, err := myInstance.GetMyInstance()
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.Success(ctx, result)
}
