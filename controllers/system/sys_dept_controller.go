package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/services/system"
	"net/http"
	"strconv"
)

type SysDeptController struct {
	controllers.BaseController
}

// AddHandler 添加部门
func (s *SysDeptController) AddHandler(ctx *gin.Context) {
	deptRequest := api.AddDeptRequest{}
	if err := ctx.ShouldBindJSON(&deptRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	deptRequest.CreateBy = s.GetUserId(ctx)
	deptRequest.UpdateBy = s.GetUserId(ctx)

	service := system.DeptService{}
	if err := service.Add(&deptRequest); err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// EditHandler 编辑部门
func (s *SysDeptController) EditHandler(ctx *gin.Context) {
	request := api.EditDeptRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	request.UpdateBy = s.GetUserId(ctx)

	service := system.DeptService{}
	if err := service.Edit(&request); err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// GetTreeHandler 树形结构查询
func (s *SysDeptController) GetTreeHandler(ctx *gin.Context) {
	service := system.DeptService{}
	all, err := service.GetTree()
	if err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	s.Success(ctx, all)
}

// ListHandler 列表查询
func (s *SysDeptController) ListHandler(ctx *gin.Context) {
	request := api.QueryDeptRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := system.DeptService{}
	all, err := service.List(&request)
	if err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	s.Success(ctx, all)
}

// RemoveHandler 删除
func (s *SysDeptController) RemoveHandler(ctx *gin.Context) {
	id := ctx.Param("id")
	deptId, _ := strconv.Atoi(id)
	service := system.DeptService{}
	if err := service.Delete(deptId); err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.JustSuccess(ctx)
}
