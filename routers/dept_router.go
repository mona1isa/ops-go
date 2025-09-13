package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/system"
)

type DeptRouter struct{}

func (*DeptRouter) Setup(r *gin.RouterGroup) {
	dc := system.SysDeptController{}
	deptGroup := r.Group("/dept")
	{
		deptGroup.POST("/add", dc.AddHandler)
		deptGroup.POST("/edit", dc.EditHandler)
		deptGroup.GET("/getTree", dc.GetTreeHandler)
		deptGroup.POST("/list", dc.ListHandler)
		deptGroup.DELETE("/:id", dc.RemoveHandler)
	}
}
