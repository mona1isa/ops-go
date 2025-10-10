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

// AddKeyHandler 添加key
func (k *KeysController) AddKeyHandler(ctx *gin.Context) {
	request := api.AddKeysRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		k.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

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
	if err := ctx.ShouldBindQuery(&request); err != nil {
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
