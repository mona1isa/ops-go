package utils

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"strconv"
	"time"
)

type HostInfo struct {
	Ip          string
	Port        int
	User        string
	Credentials string
	Type        int
}

// TestConnect 测试SSH连通性
func TestConnect(info *HostInfo) error {
	// 解密凭证（密码和密钥都可能加密存储）
	credentials, err := DecryptKey(info.Credentials)
	if err != nil {
		log.Printf("解密凭证失败: %v", err)
		return fmt.Errorf("解密凭证失败: %w", err)
	}

	// 配置 SSH 客户端
	var authMethods []ssh.AuthMethod
	if info.Type == 2 {
		// 密钥认证
		signer, err := ssh.ParsePrivateKey([]byte(credentials))
		if err != nil {
			return fmt.Errorf("解析 SSH 密钥失败: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		// 密码认证
		authMethods = append(authMethods, ssh.Password(credentials))
	}

	config := &ssh.ClientConfig{
		User:            info.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 忽略主机密钥验证（测试用）
		Timeout:         5 * time.Second,             // 设置连接超时
	}

	// 组合主机地址
	addr := net.JoinHostPort(info.Ip, strconv.Itoa(info.Port))

	// 建立TCP连接
	conn, err := net.DialTimeout("tcp", addr, config.Timeout)
	if err != nil {
		return fmt.Errorf("TCP 连接失败: %w", err)
	}
	defer conn.Close()

	// 建立SSH连接
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer sshConn.Close()

	// 初始化SSH 客户端
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()
	fmt.Println("SSH 连接成功")
	return nil
}
