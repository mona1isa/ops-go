package api

// SftpListRequest SFTP 列出目录请求
type SftpListRequest struct {
	InstanceId int    `json:"instanceId" binding:"required"`
	KeyId      int    `json:"keyId" binding:"required"`
	Path       string `json:"path"` // 默认为 "/"
}

// SftpDownloadRequest SFTP 下载文件请求
type SftpDownloadRequest struct {
	InstanceId int    `json:"instanceId" binding:"required"`
	KeyId      int    `json:"keyId" binding:"required"`
	RemotePath string `json:"remotePath" binding:"required"`
}

// SftpUploadRequest SFTP 上传文件请求
type SftpUploadRequest struct {
	InstanceId int    `json:"instanceId" binding:"required"`
	KeyId      int    `json:"keyId" binding:"required"`
	RemotePath string `json:"remotePath" binding:"required"`
}

// SftpRemoveRequest SFTP 删除文件/目录请求
type SftpRemoveRequest struct {
	InstanceId int    `json:"instanceId" binding:"required"`
	KeyId      int    `json:"keyId" binding:"required"`
	RemotePath string `json:"remotePath" binding:"required"`
	IsDir      bool   `json:"isDir"`
}

// SftpRenameRequest SFTP 重命名请求
type SftpRenameRequest struct {
	InstanceId int    `json:"instanceId" binding:"required"`
	KeyId      int    `json:"keyId" binding:"required"`
	OldPath    string `json:"oldPath" binding:"required"`
	NewPath    string `json:"newPath" binding:"required"`
}

// SftpMkdirRequest SFTP 创建目录请求
type SftpMkdirRequest struct {
	InstanceId int    `json:"instanceId" binding:"required"`
	KeyId      int    `json:"keyId" binding:"required"`
	RemotePath string `json:"remotePath" binding:"required"`
}

// SftpUploadCheckRequest SFTP 断点续传查询请求
type SftpUploadCheckRequest struct {
	InstanceId int    `json:"instanceId" binding:"required"`
	KeyId      int    `json:"keyId" binding:"required"`
	RemotePath string `json:"remotePath" binding:"required"`
	FileName   string `json:"fileName" binding:"required"`
}
