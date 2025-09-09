package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/system"
)

type MenuRouter struct {
}

func (*MenuRouter) Setup(r *gin.RouterGroup) {
	mc := system.SysMenuController{}
	menuGroup := r.Group("/menu")
	{
		menuGroup.GET("/getRoutes", mc.RouteHandler)
		menuGroup.POST("/add", mc.Add)
		menuGroup.POST("/edit", mc.Edit)
		menuGroup.POST("/list", mc.List)
		menuGroup.DELETE("/:id", mc.Remove)
	}
}
