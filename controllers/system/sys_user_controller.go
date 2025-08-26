package system

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/system/request"
	"github.com/zhany/ops-go/services/system"
	"net/http"
	"time"
)

type SysUserController struct {
	controllers.BaseController
}

// LoginHandler 登录
func (u *SysUserController) LoginHandler(c *gin.Context) {
	loginRequest := request.LoginRequest{}
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

// AddUserHandler 添加用户
func (u *SysUserController) AddUserHandler(c *gin.Context) {
	userRequest := request.UserRequest{}
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userService := system.UserService{}
	if err := userService.AddUser(userRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	result := map[string]any{
		"code": 200,
		"msg":  "success",
	}
	c.JSON(http.StatusOK, result)
}

// EditUserHandler 编辑用户
func (s *SysUserController) EditUserHandler(ctx *gin.Context) {
	userRequest := request.EditUserRequest{}
	if err := ctx.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err)
	}

	clientIp := ctx.Request.Header.Get("X-Forwarded-For")
	userRequest.LoginIP = clientIp
	userRequest.LoginDate = time.Now()

	service := system.UserService{}
	if err := service.EditUser(userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err)
		return
	}
	result := map[string]any{
		"code": 200,
		"msg":  "success",
	}
	s.Success(ctx, result)
}

func (s *SysUserController) Page(c *gin.Context) {
	userRequest := request.PageUserRequest{}
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(c, http.StatusBadRequest, err)
		return
	}
	service := system.UserService{}
	all, err := service.Page(&userRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	result := map[string]any{
		"code": 200,
		"msg":  "success",
		"data": all,
	}
	c.JSON(http.StatusOK, result)
}

func (*SysUserController) Delete(c *gin.Context) {
	service := system.UserService{}
	id := c.Param("id")
	if err := service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	result := map[string]any{
		"code": 200,
		"msg":  "success",
	}
	c.JSON(http.StatusOK, result)
}
