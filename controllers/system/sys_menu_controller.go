package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/services/system"
	"net/http"
	"strconv"
)

type SysMenuController struct {
	controllers.BaseController
}

// Add 添加菜单
func (s *SysMenuController) Add(ctx *gin.Context) {
	menu := api.AddMenuRequest{}
	if err := ctx.ShouldBindJSON(&menu); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	menu.CreateBy = s.GetUserId(ctx)
	menu.UpdateBy = s.GetUserId(ctx)

	service := system.MenuService{}
	if err := service.Add(&menu); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	s.JustSuccess(ctx)
}

// RouteHandler 前段获取菜单路由
func (s *SysMenuController) RouteHandler(ctx *gin.Context) {
	service := system.MenuService{}
	menus, err := service.RoutesList()
	if err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	s.Success(ctx, menus)
}

// List 菜单列表
func (s *SysMenuController) List(ctx *gin.Context) {
	menuListRequest := api.MenuListRequest{}
	if err := ctx.ShouldBindJSON(&menuListRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	service := system.MenuService{}
	menus, err := service.List(&menuListRequest)
	if err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	s.Success(ctx, menus)
}

// Edit 编辑菜单
func (s *SysMenuController) Edit(ctx *gin.Context) {

}

// Remove 删除菜单
func (s *SysMenuController) Remove(ctx *gin.Context) {
	id := ctx.Param("id")
	menuId, _ := strconv.Atoi(id)
	service := system.MenuService{}
	err := service.Delete(menuId)
	if err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.JustSuccess(ctx)
}
