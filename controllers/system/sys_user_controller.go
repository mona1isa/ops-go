package system

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/services/system"
	"net/http"
	"time"
)

type SysUserController struct {
	controllers.BaseController
}

// LoginHandler 登录
func (s *SysUserController) LoginHandler(c *gin.Context) {
	loginRequest := api.LoginRequest{}
	err := c.ShouldBindJSON(&loginRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置登录IP和时间
	loginRequest.LoginIP = c.ClientIP()
	now := time.Now()
	loginRequest.LoginDate = &now

	service := system.UserService{}
	jwt, err := service.UserLogin(loginRequest)
	if err != nil {
		msg := gin.H{
			"code": http.StatusBadRequest,
			"msg":  err.Error(),
		}
		c.JSON(http.StatusBadRequest, msg)
		return
	}
	result := map[string]any{
		"code":  200,
		"msg":   "success",
		"token": jwt,
	}
	c.JSON(http.StatusOK, result)
}

// LogOutHandler 登出
func (s *SysUserController) LogOutHandler(ctx *gin.Context) {
	service := system.UserService{}
	tokenString := ctx.GetHeader("Authorization")
	service.LogOut(tokenString)
	s.JustSuccess(ctx)
}

// AddUserHandler 添加用户
func (s *SysUserController) AddUserHandler(ctx *gin.Context) {
	userRequest := api.UserRequest{}
	if err := ctx.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userService := system.UserService{}
	if err := userService.AddUser(userRequest); err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// UserInfoHandler 获取用户信息
func (s *SysUserController) UserInfoHandler(ctx *gin.Context) {
	userId := s.GetUserId(ctx)
	userService := system.UserService{}
	userInfo, err := userService.GetUserInfo(userId)
	if err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.Success(ctx, userInfo)
}

// EditUserHandler 编辑用户
func (s *SysUserController) EditUserHandler(ctx *gin.Context) {
	userRequest := api.EditUserRequest{}
	if err := ctx.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if userRequest.Id <= 0 {
		s.Failure(ctx, http.StatusBadRequest, errors.New("id is required"))
		return
	}

	service := system.UserService{}
	if err := service.EditUser(userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

func (s *SysUserController) Page(c *gin.Context) {
	userRequest := api.PageUserRequest{}
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(c, http.StatusBadRequest, err)
		return
	}
	service := system.UserService{}
	all, err := service.Page(&userRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	s.Success(c, all)
}

func (s *SysUserController) Delete(ctx *gin.Context) {
	service := system.UserService{}
	id := ctx.Param("id")
	if err := service.Delete(id); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
	}
	s.JustSuccess(ctx)
}

// ChangeStatus 修改用户状态
func (s *SysUserController) ChangeStatus(ctx *gin.Context) {
	userStatus := api.UserStatusRequest{}
	if err := ctx.ShouldBindJSON(&userStatus); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	id := userStatus.Id
	if id == controllers.ADMIN_USER_ID {
		s.Failure(ctx, http.StatusBadRequest, "不能修改管理员状态")
		return
	}

	service := system.UserService{}
	if err := service.ChangeStatus(userStatus); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.JustSuccess(ctx)
}
