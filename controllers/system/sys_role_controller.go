package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/system/request"
	"github.com/zhany/ops-go/services/system"
	"net/http"
)

type SysRoleController struct {
	controllers.BaseController
}

// Add 添加角色
func (s *SysRoleController) Add(ctx *gin.Context) {
	roleRequest := request.RoleRequest{}
	if err := ctx.ShouldBindJSON(&roleRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := system.RoleService{}
	if err := service.Add(&roleRequest); err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// Edit 编辑角色
func (s *SysRoleController) Edit(ctx *gin.Context) {

}

// Page 分页查询角色
func (s *SysRoleController) Page(ctx *gin.Context) {

}

// Remove 删除角色
func (s *SysRoleController) Remove(ctx *gin.Context) {

}
