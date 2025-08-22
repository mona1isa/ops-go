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
	c.JSONP(http.StatusOK, gin.H{"status": "ok"})
}
