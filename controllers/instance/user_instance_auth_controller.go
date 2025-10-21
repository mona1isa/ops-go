package instance

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/services/instance"
	"net/http"
)

type UserInstanceAuthController struct {
	controllers.BaseController
}

// AddHandler 添加用户主机/主机分组授权
func (auth *UserInstanceAuthController) AddHandler(ctx *gin.Context) {
	userInstanceAuth := new(instance.UserInstanceAuth)
	if err := ctx.ShouldBindJSON(&userInstanceAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	err := userInstanceAuth.CreateUserInstanceAuth()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	auth.JustSuccess(ctx)
}

// DeleteHandler 删除用户主机/主机分组授权
func (auth *UserInstanceAuthController) DeleteHandler(ctx *gin.Context) {
	userInstanceAuth := new(instance.UserInstanceAuth)
	if err := ctx.ShouldBindJSON(&userInstanceAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	err := userInstanceAuth.DeleteUserInstanceAuth()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.JustSuccess(ctx)
}

// UserInstanceAuthHandler 获取用户主机/主机分组授权
func (auth *UserInstanceAuthController) UserInstanceAuthHandler(ctx *gin.Context) {
	userInstanceAuth := new(instance.UserInstanceAuth)
	if err := ctx.ShouldBindJSON(&userInstanceAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	instances, err := userInstanceAuth.GetUserInstanceAuth()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.Success(ctx, instances)
}

// ListInstanceHandler 获取用户实例列表
func (auth *UserInstanceAuthController) ListInstanceHandler(ctx *gin.Context) {
	userInstanceAuth := new(instance.UserInstanceAuth)
	if err := ctx.ShouldBindJSON(&userInstanceAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	instances, err := userInstanceAuth.GetUserInstances()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.Success(ctx, instances)
}

// PageUserInstancesHandler 分页获取用户实例列表
func (auth *UserInstanceAuthController) PageUserInstancesHandler(ctx *gin.Context) {
	userInstanceAuth := new(instance.PageUserInstanceAuth)
	if err := ctx.ShouldBindJSON(&userInstanceAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	instances, err := userInstanceAuth.GetUserInstancesPage()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.Success(ctx, instances)
}

// PageUserGroupHandler 分页获取已授权给用户的主机分组
func (auth *UserInstanceAuthController) PageUserGroupHandler(ctx *gin.Context) {
	userInstanceAuth := new(instance.PageUserInstanceAuth)
	if err := ctx.ShouldBindJSON(&userInstanceAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	groups, err := userInstanceAuth.GetUserGroupsPage()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.Success(ctx, groups)
}

// AvailableInstancesHandler 获取未绑定的主机列表
func (auth *UserInstanceAuthController) AvailableInstancesHandler(ctx *gin.Context) {
	userInstanceAuth := new(instance.PageUserInstanceAuth)
	if err := ctx.ShouldBindJSON(&userInstanceAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	instances, err := userInstanceAuth.GetInstances()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.Success(ctx, instances)
}

// AvailableGroupsHandler 获取未绑定的主机分组列表
func (auth *UserInstanceAuthController) AvailableGroupsHandler(ctx *gin.Context) {
	userInstanceAuth := new(instance.PageUserInstanceAuth)
	if err := ctx.ShouldBindJSON(&userInstanceAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	groups, err := userInstanceAuth.GetGroups()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.Success(ctx, groups)
}

func (auth *UserInstanceAuthController) AvailableKeysHandler(ctx *gin.Context) {
	var userInstanceKey instance.UserInstanceKey
	if err := ctx.ShouldBindJSON(&userInstanceKey); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	keys, err := userInstanceKey.GetUserInstanceKeys()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.Success(ctx, keys)
}

func (auth *UserInstanceAuthController) CreateUserInstanceKeyAuthHandler(ctx *gin.Context) {
	var userInstanceKeyAuth instance.UserInstanceKeyAuth
	if err := ctx.ShouldBindJSON(&userInstanceKeyAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	err := userInstanceKeyAuth.CreateUserInstanceKeyAuth()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.JustSuccess(ctx)
}

// DeleteUserInstanceKeyAuth 删除用户主机/主机分组授权
func (auth *UserInstanceAuthController) DeleteUserInstanceKeyAuthHandler(ctx *gin.Context) {
	var userInstanceKeyAuth instance.UserInstanceKeyAuth
	if err := ctx.ShouldBindJSON(&userInstanceKeyAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := userInstanceKeyAuth.DeleteUserInstanceKeyAuth(); err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err)
		return
	}
	auth.JustSuccess(ctx)
}

func (auth *UserInstanceAuthController) GetUserInstanceKeyAuthHandler(ctx *gin.Context) {
	var userInstanceKeyAuth instance.UserInstanceKeyAuth
	if err := ctx.ShouldBindJSON(&userInstanceKeyAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	result, err := userInstanceKeyAuth.GetUserInstanceKeyAuth()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err)
		return
	}
	auth.Success(ctx, result)
}

func (auth *UserInstanceAuthController) MultiKeyAuthCancelHandler(ctx *gin.Context) {
	var multiKeyAuthCancel instance.MultiKeyAuthCancel
	if err := ctx.ShouldBindJSON(&multiKeyAuthCancel); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := multiKeyAuthCancel.MultiKeyAuthCancelService(); err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err)
		return
	}
	auth.JustSuccess(ctx)
}

// GroupAvailableKeyHandler 获取主机分组可授权凭证
func (auth *UserInstanceAuthController) GroupAvailableKeyHandler(ctx *gin.Context) {
	var groupAuth instance.InstanceGroupUserKeyAuth
	if err := ctx.ShouldBindJSON(&groupAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	keys, err := groupAuth.GroupAvailableKeyService()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.Success(ctx, keys)
}

// 主机分组授权凭证
func (auth *UserInstanceAuthController) GroupAuthKeyHandler(ctx *gin.Context) {
	var groupAuth instance.InstanceGroupUserKeyAuth
	if err := ctx.ShouldBindJSON(&groupAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	err := groupAuth.GroupAuthKeyService()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.JustSuccess(ctx)
}

// 主机分组解除授权凭证
func (auth *UserInstanceAuthController) GroupCancelAuthKeyHandler(ctx *gin.Context) {
	var groupAuth instance.InstanceGroupUserKeyAuth
	if err := ctx.ShouldBindJSON(&groupAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	err := groupAuth.GroupCancelAuthKeyService()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.JustSuccess(ctx)
}

func (auth *UserInstanceAuthController) GroupCancelAuthKeyBatchHandler(ctx *gin.Context) {
	var groupAuth instance.InstanceGroupUserKeyAuthMulti
	if err := ctx.ShouldBindJSON(&groupAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	err := groupAuth.MultiGroupCancelAuthKeyService()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.JustSuccess(ctx)
}

func (auth *UserInstanceAuthController) GetInstanceGroupAuthedKeyHandler(ctx *gin.Context) {
	var groupAuth instance.InstanceGroupUserKeyAuth
	if err := ctx.ShouldBindJSON(&groupAuth); err != nil {
		auth.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	keys, err := groupAuth.GroupAuthedKeyService()
	if err != nil {
		auth.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	auth.Success(ctx, keys)
}
