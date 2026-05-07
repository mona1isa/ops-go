package instance

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/instance"
	"github.com/zhany/ops-go/utils"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

// SftpController SFTP 文件管理控制器
type SftpController struct {
	controllers.BaseController
}

// SftpFileInfo SFTP 文件信息响应
type SftpFileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`
	IsDir   bool   `json:"isDir"`
}

// validateAndConnect 验证权限并建立 SFTP 连接
func (c *SftpController) validateAndConnect(ctx *gin.Context, instanceId, keyId int) (*utils.SftpClient, *models.OpsInstance, error) {
	userIdStr := c.GetUserId(ctx)
	userId, _ := strconv.Atoi(userIdStr)
	isAdmin := c.IsAdminUser(ctx)

	// 验证主机是否存在
	var opsInstance models.OpsInstance
	if err := models.DB.First(&opsInstance, instanceId).Error; err != nil {
		return nil, nil, fmt.Errorf("主机不存在")
	}

	if opsInstance.Status != "1" {
		return nil, nil, fmt.Errorf("主机当前状态异常，无法访问")
	}

	// 非管理员需要校验用户-主机-凭证授权关系
	if !isAdmin {
		keyAuthService := &instance.UserInstanceKeyAuth{
			UserId:     userId,
			InstanceId: instanceId,
			AuthType:   1,
		}
		keys, err := keyAuthService.GetUserInstanceKeyAuth()
		if err != nil {
			return nil, nil, fmt.Errorf("获取用户凭证授权失败: %w", err)
		}
		if len(keys) == 0 {
			return nil, nil, fmt.Errorf("您没有该主机的登录凭证权限")
		}
		// 校验指定的 keyId 是否在授权列表中
		found := false
		for _, key := range keys {
			if int(key.ID) == keyId {
				found = true
				break
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("您没有使用该凭证的权限")
		}
	}

	// 获取凭证信息
	var key models.OpsKey
	if err := models.DB.First(&key, keyId).Error; err != nil {
		return nil, nil, fmt.Errorf("凭证不存在")
	}

	// 获取明文凭证（密码和密钥都可能加密存储）
	credentials, err := utils.DecryptKey(key.Credentials)
	if err != nil {
		log.Printf("解密凭证失败: %v", err)
		return nil, nil, fmt.Errorf("解密凭证失败")
	}

	// 建立 SFTP 连接
	hostInfo := &utils.HostInfo{
		Ip:          opsInstance.Ip,
		Port:        key.Port,
		User:        key.User,
		Credentials: credentials,
		Type:        key.Type,
	}

	sftpClient, err := utils.NewSftpClient(hostInfo)
	if err != nil {
		return nil, nil, err
	}

	return sftpClient, &opsInstance, nil
}

// ListHandler 列出远程目录内容
func (c *SftpController) ListHandler(ctx *gin.Context) {
	var request api.SftpListRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	sftpClient, _, err := c.validateAndConnect(ctx, request.InstanceId, request.KeyId)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	defer sftpClient.Close()

	// 路径处理：空字符串表示用户 home 目录
	remotePath := request.Path
	if remotePath == "" {
		// 获取当前工作目录（即用户 home 目录）
		wd, err := sftpClient.Client.Getwd()
		if err != nil {
			c.Failure(ctx, http.StatusBadRequest, "获取当前目录失败: "+err.Error())
			return
		}
		remotePath = wd
	} else {
		remotePath = path.Clean(remotePath)
	}

	// 读取目录
	entries, err := sftpClient.Client.ReadDir(remotePath)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "读取目录失败: "+err.Error())
		return
	}

	var files []SftpFileInfo
	for _, entry := range entries {
		// 跳过隐藏文件（以 . 开头）可根据需求调整
		files = append(files, SftpFileInfo{
			Name:    entry.Name(),
			Size:    entry.Size(),
			Mode:    entry.Mode().String(),
			ModTime: entry.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   entry.IsDir(),
		})
	}

	c.Success(ctx, gin.H{
		"path":  remotePath,
		"files": files,
	})
}

// DownloadHandler 下载远程文件
func (c *SftpController) DownloadHandler(ctx *gin.Context) {
	var request api.SftpDownloadRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	remotePath := path.Clean(request.RemotePath)

	sftpClient, _, err := c.validateAndConnect(ctx, request.InstanceId, request.KeyId)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	defer sftpClient.Close()

	// 打开远程文件
	remoteFile, err := sftpClient.Client.Open(remotePath)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "打开远程文件失败: "+err.Error())
		return
	}
	defer remoteFile.Close()

	// 获取文件信息
	stat, err := remoteFile.Stat()
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "获取文件信息失败: "+err.Error())
		return
	}

	if stat.IsDir() {
		c.Failure(ctx, http.StatusBadRequest, "不能下载目录")
		return
	}

	// 设置响应头
	filename := filepath.Base(remotePath)
	ctx.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	ctx.Writer.Header().Set("Content-Type", "application/octet-stream")
	ctx.Writer.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))

	// 流式传输文件
	_, err = io.Copy(ctx.Writer, remoteFile)
	if err != nil {
		log.Printf("下载文件流式传输失败: %v", err)
	}
}

// UploadHandler 上传文件到远程目录
func (c *SftpController) UploadHandler(ctx *gin.Context) {
	instanceIdStr := ctx.PostForm("instanceId")
	keyIdStr := ctx.PostForm("keyId")
	remotePath := ctx.PostForm("remotePath")

	instanceId, err := strconv.Atoi(instanceIdStr)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "主机ID格式错误")
		return
	}
	keyId, err := strconv.Atoi(keyIdStr)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "凭证ID格式错误")
		return
	}

	remotePath = path.Clean(remotePath)
	if remotePath == "" {
		remotePath = "/"
	}

	// 获取上传的文件
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "获取上传文件失败: "+err.Error())
		return
	}

	sftpClient, _, err := c.validateAndConnect(ctx, instanceId, keyId)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	defer sftpClient.Close()

	// 打开本地上传的临时文件
	uploadedFile, err := fileHeader.Open()
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "读取上传文件失败: "+err.Error())
		return
	}
	defer uploadedFile.Close()

	// 组合远程文件路径
	remoteFilePath := path.Join(remotePath, fileHeader.Filename)

	// 创建远程文件
	remoteFile, err := sftpClient.Client.Create(remoteFilePath)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "创建远程文件失败: "+err.Error())
		return
	}
	defer remoteFile.Close()

	// 写入数据
	_, err = io.Copy(remoteFile, uploadedFile)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "写入远程文件失败: "+err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// RemoveHandler 删除远程文件/目录
func (c *SftpController) RemoveHandler(ctx *gin.Context) {
	var request api.SftpRemoveRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	remotePath := path.Clean(request.RemotePath)

	sftpClient, _, err := c.validateAndConnect(ctx, request.InstanceId, request.KeyId)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	defer sftpClient.Close()

	if request.IsDir {
		err = sftpClient.Client.RemoveDirectory(remotePath)
	} else {
		err = sftpClient.Client.Remove(remotePath)
	}

	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "删除失败: "+err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// RenameHandler 重命名远程文件/目录
func (c *SftpController) RenameHandler(ctx *gin.Context) {
	var request api.SftpRenameRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	oldPath := path.Clean(request.OldPath)
	newPath := path.Clean(request.NewPath)

	sftpClient, _, err := c.validateAndConnect(ctx, request.InstanceId, request.KeyId)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	defer sftpClient.Close()

	if err := sftpClient.Client.Rename(oldPath, newPath); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "重命名失败: "+err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// MkdirHandler 创建远程目录
func (c *SftpController) MkdirHandler(ctx *gin.Context) {
	var request api.SftpMkdirRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	remotePath := path.Clean(request.RemotePath)

	sftpClient, _, err := c.validateAndConnect(ctx, request.InstanceId, request.KeyId)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	defer sftpClient.Close()

	if err := sftpClient.Client.MkdirAll(remotePath); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "创建目录失败: "+err.Error())
		return
	}

	c.JustSuccess(ctx)
}

// SftpUploadRequest 上传请求结构体（用于文档生成）
// Deprecated: 仅用于保持结构体定义，实际上传使用 multipart/form-data
func (c *SftpController) SftpUploadRequest(ctx *gin.Context) {
	c.Failure(ctx, http.StatusMethodNotAllowed, "请使用 multipart/form-data 上传文件")
}

// init 用于确保导入的包被使用
func init() {
	_ = os.Getenv("")
	_ = time.Now()
}
