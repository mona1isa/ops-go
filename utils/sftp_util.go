package utils

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"strconv"
	"time"
)

// SftpClient SFTP 客户端封装
type SftpClient struct {
	Client *sftp.Client
	sshConn *ssh.Client
}

// NewSftpClient 创建新的 SFTP 客户端
// 复用现有 SSH 认证逻辑（密码/密钥、解密凭证、端口默认 22）
func NewSftpClient(info *HostInfo) (*SftpClient, error) {
	// 如果是密码类型，解密凭证
	credentials := info.Credentials
	if info.Type == 1 {
		decrypted, err := DecryptKey(credentials)
		if err != nil {
			log.Printf("解密凭证失败: %v", err)
			return nil, fmt.Errorf("解密凭证失败: %w", err)
		}
		credentials = decrypted
	}

	// 配置 SSH 客户端
	var authMethods []ssh.AuthMethod
	if info.Type == 2 {
		// 密钥认证
		signer, err := ssh.ParsePrivateKey([]byte(credentials))
		if err != nil {
			return nil, fmt.Errorf("解析 SSH 密钥失败: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		// 密码认证
		authMethods = append(authMethods, ssh.Password(credentials))
	}

	config := &ssh.ClientConfig{
		User:            info.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// 组合主机地址
	port := info.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(info.Ip, strconv.Itoa(port))

	// 建立 SSH 连接
	sshConn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}

	// 创建 SFTP 客户端
	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("创建 SFTP 客户端失败: %w", err)
	}

	return &SftpClient{
		Client:  sftpClient,
		sshConn: sshConn,
	}, nil
}

// Close 关闭 SFTP 客户端和 SSH 连接
func (s *SftpClient) Close() {
	if s.Client != nil {
		if err := s.Client.Close(); err != nil {
			log.Printf("关闭 SFTP 客户端失败: %v", err)
		}
	}
	if s.sshConn != nil {
		if err := s.sshConn.Close(); err != nil {
			log.Printf("关闭 SSH 连接失败: %v", err)
		}
	}
}
