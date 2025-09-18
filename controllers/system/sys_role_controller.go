package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/system"
	"net/http"
	"strconv"
)

type SysRoleController struct {
	controllers.BaseController
}

// Add 添加角色
func (s *SysRoleController) Add(ctx *gin.Context) {
	roleRequest := api.RoleRequest{}
	if err := ctx.ShouldBindJSON(&roleRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := system.RoleService{}
	if err := service.Add(&roleRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// Edit 编辑角色
func (s *SysRoleController) Edit(ctx *gin.Context) {
	editRoleRequest := api.EditRoleRequest{}
	if err := ctx.ShouldBindJSON(&editRoleRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := system.RoleService{}
	if err := service.Edit(&editRoleRequest); err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// List 角色列表
func (s *SysRoleController) List(ctx *gin.Context) {
	service := system.RoleService{}
	roles, err := service.List()
	if err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.Success(ctx, roles)
}

// Page 分页查询角色
func (s *SysRoleController) Page(ctx *gin.Context) {
	pageRoleRequest := api.PageRoleRequest{}
	if err := ctx.ShouldBindJSON(&pageRoleRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := system.RoleService{}
	page, err := service.Page(&pageRoleRequest)
	if err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.PageSuccess(ctx, page.Data, page.Total, page.TotalPage, pageRoleRequest.PageNum, pageRoleRequest.PageSize)
}

// Remove 删除角色
func (s *SysRoleController) Remove(ctx *gin.Context) {
	roleId := ctx.Param("id")
	id, _ := strconv.Atoi(roleId)

	var count int64
	if err := config.DB.Model(models.SysUserRole{}).Where("role_id = ?", id).Count(&count).Error; err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if count > 0 {
		s.Failure(ctx, http.StatusBadRequest, "该角色下有用户，无法删除")
		return
	}

	service := system.RoleService{}
	if err := service.Remove(id); err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.JustSuccess(ctx)
}
