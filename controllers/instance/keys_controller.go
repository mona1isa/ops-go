package instance

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/services/instance"
	"net/http"
	"strconv"
)

type KeysController struct {
	controllers.BaseController
}

// ListHandler 获取key列表
func (k *KeysController) ListHandler(ctx *gin.Context) {
	service := instance.KeysService{}
	info, err := service.ListKeys()
	if err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.Success(ctx, info)
}

// AddKeyHandler 添加key
func (k *KeysController) AddKeyHandler(ctx *gin.Context) {
	request := api.AddKeysRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		k.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := k.GetUserId(ctx)
	request.CreateBy = userId
	request.UpdateBy = userId

	service := instance.KeysService{}
	if err := service.AddKey(request); err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.JustSuccess(ctx)
}

// EditKeyHandler 编辑key
func (k *KeysController) EditKeyHandler(ctx *gin.Context) {
	request := api.UpdateKeysRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		k.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.KeysService{}
	if err := service.EditKey(request); err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.JustSuccess(ctx)
}

// PageKeyHandler 分页查询key
func (k *KeysController) PageKeyHandler(ctx *gin.Context) {
	request := api.PageKeysRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		k.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.KeysService{}
	info, err := service.PageKey(request)
	if err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.Success(ctx, info)
}

// ChangeStatusHandler 修改key状态
func (k *KeysController) ChangeStatusHandler(ctx *gin.Context) {
	request := api.ChangeStatusRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		k.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := instance.KeysService{}
	if err := service.ChangeStatus(request); err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.JustSuccess(ctx)
}

// DeleteKeyHandler 删除key
func (k *KeysController) DeleteKeyHandler(ctx *gin.Context) {
	id := ctx.Param("id")
	keyId, _ := strconv.Atoi(id)

	service := instance.KeysService{}
	if err := service.DeleteKey(keyId); err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.JustSuccess(ctx)
}

// AvailableKeysHandler 获取可用于绑定该主机的key
func (k *KeysController) AvailableKeysHandler(ctx *gin.Context) {
	instanceId := ctx.Param("instanceId")
	instanceIdInt, _ := strconv.Atoi(instanceId)

	service := instance.KeysService{}
	info, err := service.AvailableKeys(instanceIdInt)
	if err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.Success(ctx, info)
}

// AvailableKeysBySystemHandler 获取可用于绑定该主机的key(根据系统类型过滤)
func (k *KeysController) AvailableKeysBySystemHandler(ctx *gin.Context) {
	request := api.OsTypeRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		k.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := instance.KeysService{}
	info, err := service.AvailableKeysBySystem(request)
	if err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.Success(ctx, info)
}

// GetPublicKeyHandler 获取公钥用于加密凭证
func (k *KeysController) GetPublicKeyHandler(ctx *gin.Context) {
	service := instance.KeysService{}
	publicKey, err := service.GetPublicKey()
	if err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.Success(ctx, gin.H{
		"publicKey": publicKey,
	})
}

// GetKeyDetailHandler 获取凭证详情
func (k *KeysController) GetKeyDetailHandler(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		k.Failure(ctx, http.StatusBadRequest, "无效的凭证ID")
		return
	}
	service := instance.KeysService{}
	key, err := service.GetKeyDetail(id)
	if err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.Success(ctx, key)
}

// GetKeyInstancesHandler 获取凭证绑定的主机列表（分页）
func (k *KeysController) GetKeyInstancesHandler(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		k.Failure(ctx, http.StatusBadRequest, "无效的凭证ID")
		return
	}
	pageNum, _ := strconv.Atoi(ctx.DefaultQuery("pageNum", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))
	service := instance.KeysService{}
	result, err := service.GetKeyInstances(id, pageNum, pageSize)
	if err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.Success(ctx, result)
}

// GetAvailableInstancesHandler 获取未绑定该凭证的主机列表
func (k *KeysController) GetAvailableInstancesHandler(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		k.Failure(ctx, http.StatusBadRequest, "无效的凭证ID")
		return
	}
	query := api.AvailableInstancesQuery{}
	_ = ctx.ShouldBindQuery(&query)

	service := instance.KeysService{}
	instances, err := service.GetAvailableInstances(id, query.Name, query.Ip)
	if err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.Success(ctx, instances)
}

// BindInstancesHandler 批量绑定主机到凭证
func (k *KeysController) BindInstancesHandler(ctx *gin.Context) {
	request := api.BindInstancesRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		k.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := instance.KeysService{}
	if err := service.BindInstances(request.KeyId, request.InstanceIds); err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.JustSuccess(ctx)
}

// UnbindInstanceHandler 解绑凭证下的某个主机
func (k *KeysController) UnbindInstanceHandler(ctx *gin.Context) {
	keyIdStr := ctx.Param("keyId")
	keyId, err := strconv.Atoi(keyIdStr)
	if err != nil {
		k.Failure(ctx, http.StatusBadRequest, "无效的凭证ID")
		return
	}
	instanceIdStr := ctx.Param("instanceId")
	instanceId, err := strconv.Atoi(instanceIdStr)
	if err != nil {
		k.Failure(ctx, http.StatusBadRequest, "无效的主机ID")
		return
	}
	service := instance.KeysService{}
	if err := service.UnbindInstance(keyId, instanceId); err != nil {
		k.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	k.JustSuccess(ctx)
}
