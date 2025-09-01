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
		userGroup.POST("/add", uc.AddUserHandler)
		userGroup.POST("/edit", uc.EditUserHandler)
		userGroup.POST("/page", uc.Page)
		userGroup.DELETE("/rm/:id", uc.Delete)
		userGroup.POST("/changeStatus", uc.ChangeStatus)
	}
}
