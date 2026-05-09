package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/instance"
)

type FileRouter struct{}

func (*FileRouter) Setup(r *gin.RouterGroup) {
	controller := instance.FileController{}
	group := r.Group("/file")
	{
		group.POST("/local/upload", controller.UploadLocalHandler)
		group.POST("/local/list", controller.ListLocalFilesHandler)
		group.POST("/local/delete", controller.DeleteLocalFileHandler)
	}
}
