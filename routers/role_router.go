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
		roleGroup.POST("/list", sr.List)
		roleGroup.POST("/page", sr.Page)
		roleGroup.DELETE("/:id", sr.Remove)
		roleGroup.GET("/menu/:roleId", sr.GetMenuIds)
		roleGroup.GET("/user/:roleId", sr.GetUserIds)
		roleGroup.POST("/assignUsers", sr.RoleAsignUsers)
	}
}
