package instance

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/services/instance"
)

type InstanceImportController struct {
	controllers.BaseController
}

// DownloadTemplateHandler 下载批量导入模板
func (c *InstanceImportController) DownloadTemplateHandler(ctx *gin.Context) {
	service := instance.InstanceImportService{}
	result, err := service.DownloadTemplate()
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.Success(ctx, result)
}

// BatchImportHandler 上传文件并批量导入主机
func (c *InstanceImportController) BatchImportHandler(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "请选择要导入的文件")
		return
	}
	if fileHeader.Size > instance.ImportMaxFileSize {
		c.Failure(ctx, http.StatusBadRequest, "文件超过 5MB，请拆分后重试")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "文件读取失败")
		return
	}
	defer file.Close()

	service := instance.InstanceImportService{}
	result, err := service.Import(fileHeader.Filename, file, c.GetUserId(ctx))
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	c.Success(ctx, result)
}
