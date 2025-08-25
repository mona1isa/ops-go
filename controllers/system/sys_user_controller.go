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
	service := system.UserService{}
	err = service.UserLogin(loginRequest)
	if err != nil {
		msg := gin.H{
			"code": http.StatusBadRequest,
			"msg":  err,
		}
		c.JSON(http.StatusBadRequest, msg)
		return
	}
	result := map[string]any{
		"code": 200,
		"msg":  "success",
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

	clientIp := c.Request.Header.Get("X-Forwarded-For")
	userRequest.LoginIP = clientIp
	userRequest.LoginDate = time.Now()

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
func (*SysUserController) EditUserHandler(c *gin.Context) {
	userRequest := request.EditUserRequest{}
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	clientIp := c.Request.Header.Get("X-Forwarded-For")
	userRequest.LoginIP = clientIp
	userRequest.LoginDate = time.Now()

	service := system.UserService{}
	if err := service.EditUser(userRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	result := map[string]any{
		"code": 200,
		"msg":  "success",
	}
	c.JSON(http.StatusOK, result)
}

func (*SysUserController) All(c *gin.Context) {
	service := system.UserService{}
	all, err := service.All()
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
