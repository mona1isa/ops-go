package instance

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
)

const localFileDir = "/data/ops"

// FileController 本地文件管理控制器
type FileController struct {
	controllers.BaseController
}

// LocalFileInfo 本地文件信息
type LocalFileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

// ensureDir 确保目录存在
func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// UploadLocalHandler 上传文件到本地服务器 /data/ops
func (c *FileController) UploadLocalHandler(ctx *gin.Context) {
	if err := ensureDir(localFileDir); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "创建目录失败: "+err.Error())
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "获取上传文件失败: "+err.Error())
		return
	}

	// 安全检查：防止路径遍历
	filename := filepath.Base(fileHeader.Filename)
	if filename == "" || filename == "." || filename == ".." {
		c.Failure(ctx, http.StatusBadRequest, "非法文件名")
		return
	}

	savePath := path.Join(localFileDir, filename)

	if err := ctx.SaveUploadedFile(fileHeader, savePath); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "保存文件失败: "+err.Error())
		return
	}

	c.Success(ctx, gin.H{
		"name": filename,
		"path": savePath,
	})
}

// ListLocalFilesHandler 列出 /data/ops 下的文件列表
func (c *FileController) ListLocalFilesHandler(ctx *gin.Context) {
	if err := ensureDir(localFileDir); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "创建目录失败: "+err.Error())
		return
	}

	entries, err := os.ReadDir(localFileDir)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "读取目录失败: "+err.Error())
		return
	}

	var files []LocalFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, LocalFileInfo{
			Name:    info.Name(),
			Path:    path.Join(localFileDir, info.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	c.Success(ctx, files)
}

// DeleteLocalFileHandler 删除 /data/ops 下的指定文件
func (c *FileController) DeleteLocalFileHandler(ctx *gin.Context) {
	filename := ctx.PostForm("filename")
	if filename == "" {
		c.Failure(ctx, http.StatusBadRequest, "文件名不能为空")
		return
	}

	// 安全检查
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == ".." {
		c.Failure(ctx, http.StatusBadRequest, "非法文件名")
		return
	}

	filePath := path.Join(localFileDir, filename)

	// 确保文件在 /data/ops 目录下
	absDir, _ := filepath.Abs(localFileDir)
	absFile, _ := filepath.Abs(filePath)
	if !filepath.HasPrefix(absFile, absDir) {
		c.Failure(ctx, http.StatusBadRequest, "非法文件路径")
		return
	}

	if err := os.Remove(filePath); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "删除文件失败: "+err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// init 用于确保导入的包被使用
func init() {
	_ = strconv.Itoa(0)
	_ = time.Now()
}
