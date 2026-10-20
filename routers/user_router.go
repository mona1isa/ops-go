package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/system"
)

type UserRouter struct{}

func (*UserRouter) Setup(r *gin.RouterGroup) {
	uc := system.SysUserController{}
	userGroup := r.Group("/user")
	{
		userGroup.POST("/login", uc.LoginHandler)
		userGroup.POST("/login/mfa", uc.MfaLoginHandler)
		userGroup.GET("/logout", uc.LogOutHandler)
		userGroup.GET("/mfa/status", uc.MfaStatusHandler)
		userGroup.POST("/mfa/bind/start", uc.MfaBindStartHandler)
		userGroup.POST("/mfa/bind/confirm", uc.MfaBindConfirmHandler)
		userGroup.POST("/mfa/unbind", uc.MfaUnbindHandler)
		userGroup.GET("/info", uc.UserInfoHandler)
		userGroup.POST("/add", uc.AddUserHandler)
		userGroup.POST("/updatePersonalInfo", uc.UpdatePersonalInfo)
		userGroup.POST("/updatePassword", uc.UpdatePassword)
		userGroup.POST("/edit", uc.EditUserHandler)
		userGroup.POST("/page", uc.Page)
		userGroup.DELETE("/rm/:id", uc.Delete)
		userGroup.POST("/changeStatus", uc.ChangeStatus)
	}
}
