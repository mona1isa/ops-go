package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/system"
)

type RoleRouter struct{}

func (*RoleRouter) Setup(r *gin.RouterGroup) {
	sr := system.SysRoleController{}
	roleGroup := r.Group("/role")
	{
		roleGroup.POST("/add", sr.Add)
		roleGroup.POST("/edit", sr.Edit)
		roleGroup.POST("/page", sr.Page)
		roleGroup.DELETE("/:id", sr.Remove)
	}
}
