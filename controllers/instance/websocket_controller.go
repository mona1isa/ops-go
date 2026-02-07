package instance

import (
	"errors"
	"fmt"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/middleware"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/instance"
	"github.com/zhany/ops-go/utils"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

type WebSocketController struct {
	controllers.BaseController
}

// SSHSession 管理SSH会话
type SSHSession struct {
	Conn       *ssh.Client
	Session    *ssh.Session
	StdoutPipe io.Reader
	StdinPipe  io.WriteCloser
	InstanceId int
	UserId     int
	KeyId      int
	mu         sync.Mutex
}

// 存储活跃的SSH会话，使用连接ID作为key
var (
	activeSessions = make(map[string]*SSHSession)
	sessionMutex   sync.RWMutex
)

// generateSessionID 生成会话ID
func generateSessionID(userId, instanceId int) string {
	return fmt.Sprintf("%d:%d:%d", userId, instanceId, time.Now().UnixNano())
}

// WebSocket升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中，应该检查Origin头
		// 开发环境允许所有源
		return true
	},
	// 允许的子协议
	Subprotocols: []string{},
}

// WebSocketMessage WebSocket消息格式
type WebSocketMessage struct {
	Type    string      `json:"type"`    // connect, resize, data, close
	Data    string      `json:"data"`    // 数据内容
	Rows    int         `json:"rows"`    // 终端行数
	Cols    int         `json:"cols"`    // 终端列数
	KeyId   int         `json:"keyId"`   // 凭证ID
	Payload interface{} `json:"payload"` // 其他负载
}

// WebSocketHandler WebSocket处理函数
func (c *WebSocketController) WebSocketHandler(ctx *gin.Context) {
	// 验证 JWT token（优先从 header 获取，其次从查询参数获取）
	authorization := ctx.GetHeader("Authorization")
	if authorization == "" {
		// 尝试从查询参数获取 token
		authorization = ctx.Query("token")
		if authorization == "" {
			c.sendErrorResponse(ctx.Writer, "Authorization header or token parameter is missing", http.StatusUnauthorized)
			return
		}
	}

	// 验证 token 并获取用户信息
	token, err := middleware.ValidateToken(authorization)
	if err != nil {
		c.sendErrorResponse(ctx.Writer, "Invalid Token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// 获取用户ID
	userId, err := strconv.Atoi(token.UserID)
	if err != nil {
		c.sendErrorResponse(ctx.Writer, "用户ID无效", http.StatusUnauthorized)
		return
	}

	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 获取主机ID参数（从查询参数获取）
	instanceIdStr := ctx.Query("instanceId")
	if instanceIdStr == "" {
		c.sendError(conn, "主机ID不能为空")
		return
	}
	instanceId, err := strconv.Atoi(instanceIdStr)
	if err != nil {
		c.sendError(conn, "主机ID格式错误")
		return
	}

	// 获取用户对该主机的权限凭证
	keyAuthService := &instance.UserInstanceKeyAuth{
		UserId:     userId,
		InstanceId: instanceId,
		AuthType:   1,
	}
	keys, err := keyAuthService.GetUserInstanceKeyAuth()
	if err != nil {
		c.sendError(conn, "获取凭证失败: "+err.Error())
		return
	}

	if len(keys) == 0 {
		c.sendError(conn, "您没有该主机的登录凭证权限")
		return
	}

	// 生成会话ID
	sessionID := generateSessionID(userId, instanceId)

	// 如果只有一个凭证，直接连接；否则返回凭证列表
	if len(keys) == 1 {
		// 直接连接
		if err := c.connectToInstance(conn, sessionID, userId, instanceId, keys[0]); err != nil {
			c.sendError(conn, "连接失败: "+err.Error())
			return
		}
	} else {
		// 返回凭证列表供用户选择
		c.sendCredentialsList(conn, keys)
	}

	// 处理WebSocket消息
	for {
		var msg WebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("读取WebSocket消息失败: %v", err)
			c.handleClose(sessionID)
			break
		}

		switch msg.Type {
		case "connect":
			// 用户选择凭证后连接
			if msg.KeyId == 0 {
				c.sendError(conn, "凭证ID不能为空")
				continue
			}
			var selectedKey *models.OpsKey
			for _, key := range keys {
				if key.ID == msg.KeyId {
					selectedKey = &key
					break
				}
			}
			if selectedKey == nil {
				c.sendError(conn, "无效的凭证ID")
				continue
			}
			if err := c.connectToInstance(conn, sessionID, userId, instanceId, *selectedKey); err != nil {
				c.sendError(conn, "连接失败: "+err.Error())
				continue
			}

		case "resize":
			// 处理终端大小调整
			c.handleResize(sessionID, msg)

		case "data":
			// 处理终端输入数据
			c.handleData(sessionID, msg)

		case "close":
			// 关闭连接
			c.handleClose(sessionID)
			return
		}
	}
}

// connectToInstance 连接到远程主机
func (c *WebSocketController) connectToInstance(conn *websocket.Conn, sessionID string, userId, instanceId int, key models.OpsKey) error {
	// 验证主机是否存在
	var instance models.OpsInstance
	if err := models.DB.First(&instance, instanceId).Error; err != nil {
		return errors.New("主机不存在")
	}

	// 检查主机状态
	if instance.Status != "1" {
		statusText := "stopped"
		if instance.Status == "" {
			statusText = "unknown"
		}
		return fmt.Errorf("主机 %s (%s) 当前状态：%s，无法访问", instance.Name, instance.Ip, statusText)
	}

	// 获取明文凭证
	var credentials string
	var err error
	if key.Type == 1 {
		// 密码类型，需要解密
		credentials, err = utils.DecryptKey(key.Credentials)
		if err != nil {
			log.Printf("解密凭证失败: %v", err)
			return errors.New("解密凭证失败")
		}
	} else {
		// 密钥类型，直接使用
		credentials = key.Credentials
	}

	// 建立SSH连接
	addr := net.JoinHostPort(instance.Ip, strconv.Itoa(key.Port))
	if key.Port == 0 {
		addr = net.JoinHostPort(instance.Ip, "22")
	}

	var authMethods []ssh.AuthMethod
	if key.Type == 2 {
		// 密钥认证
		signer, err := ssh.ParsePrivateKey([]byte(credentials))
		if err != nil {
			return errors.New("解析SSH密钥失败: " + err.Error())
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		// 密码认证
		authMethods = append(authMethods, ssh.Password(credentials))
	}

	sshConfig := &ssh.ClientConfig{
		User:            key.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	sshConn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return errors.New("SSH连接失败: " + err.Error())
	}

	session, err := sshConn.NewSession()
	if err != nil {
		sshConn.Close()
		return errors.New("创建SSH会话失败: " + err.Error())
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		sshConn.Close()
		return errors.New("获取标准输出失败: " + err.Error())
	}

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		session.Close()
		sshConn.Close()
		return errors.New("获取标准输入失败: " + err.Error())
	}

	// 请求伪终端
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		session.Close()
		sshConn.Close()
		return errors.New("请求伪终端失败: " + err.Error())
	}

	// 保存会话
	sshSession := &SSHSession{
		Conn:       sshConn,
		Session:    session,
		StdoutPipe: stdoutPipe,
		StdinPipe:  stdinPipe,
		InstanceId: instanceId,
		UserId:     userId,
		KeyId:      key.ID,
	}

	sessionMutex.Lock()
	activeSessions[sessionID] = sshSession
	sessionMutex.Unlock()

	// 启动SSH会话
	if err := session.Shell(); err != nil {
		session.Close()
		sshConn.Close()
		sessionMutex.Lock()
		delete(activeSessions, sessionID)
		sessionMutex.Unlock()
		return errors.New("启动Shell失败: " + err.Error())
	}

	// 发送连接成功消息
	c.sendSuccess(conn, "连接成功")

	// 启动goroutine读取SSH输出并发送到WebSocket
	go c.readSSHOutput(conn, sessionID, sshSession)

	return nil
}

// readSSHOutput 读取SSH输出并发送到WebSocket
func (c *WebSocketController) readSSHOutput(conn *websocket.Conn, sessionID string, sshSession *SSHSession) {
	defer func() {
		c.handleClose(sessionID)
	}()

	buf := make([]byte, 4096)
	for {
		n, err := sshSession.StdoutPipe.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("读取SSH输出失败: %v", err)
			}
			return
		}
		if n > 0 {
			if err := conn.WriteJSON(WebSocketMessage{
				Type: "data",
				Data: string(buf[:n]),
			}); err != nil {
				log.Printf("发送WebSocket数据失败: %v", err)
				return
			}
		}
	}
}

// handleResize 处理终端大小调整
func (c *WebSocketController) handleResize(sessionID string, msg WebSocketMessage) {
	sessionMutex.RLock()
	sshSession, ok := activeSessions[sessionID]
	sessionMutex.RUnlock()

	if !ok {
		return
	}

	sshSession.mu.Lock()
	defer sshSession.mu.Unlock()

	if err := sshSession.Session.WindowChange(msg.Rows, msg.Cols); err != nil {
		log.Printf("调整终端大小失败: %v", err)
	}
}

// handleData 处理终端输入数据
func (c *WebSocketController) handleData(sessionID string, msg WebSocketMessage) {
	sessionMutex.RLock()
	sshSession, ok := activeSessions[sessionID]
	sessionMutex.RUnlock()

	if !ok {
		return
	}

	sshSession.mu.Lock()
	defer sshSession.mu.Unlock()

	if _, err := sshSession.StdinPipe.Write([]byte(msg.Data)); err != nil {
		log.Printf("写入SSH输入失败: %v", err)
	}
}

// handleClose 关闭SSH会话
func (c *WebSocketController) handleClose(sessionID string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	if sshSession, ok := activeSessions[sessionID]; ok {
		sshSession.mu.Lock()
		// 关闭标准输入管道
		if sshSession.StdinPipe != nil {
			sshSession.StdinPipe.Close()
		}
		// 关闭SSH会话
		if sshSession.Session != nil {
			sshSession.Session.Close()
		}
		// 关闭SSH连接
		if sshSession.Conn != nil {
			sshSession.Conn.Close()
		}
		sshSession.mu.Unlock()
		delete(activeSessions, sessionID)
	}
}

// sendCredentialsList 发送凭证列表
func (c *WebSocketController) sendCredentialsList(conn *websocket.Conn, keys []models.OpsKey) {
	conn.WriteJSON(WebSocketMessage{
		Type: "credentials",
		Payload: map[string]interface{}{
			"keys": keys,
		},
	})
}

// sendError 发送错误消息
func (c *WebSocketController) sendError(conn *websocket.Conn, message string) {
	conn.WriteJSON(WebSocketMessage{
		Type: "error",
		Data: message,
	})
}

// sendSuccess 发送成功消息
func (c *WebSocketController) sendSuccess(conn *websocket.Conn, message string) {
	conn.WriteJSON(WebSocketMessage{
		Type: "success",
		Data: message,
	})
}

// sendErrorResponse 发送HTTP错误响应
func (c *WebSocketController) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(fmt.Sprintf(`{"code": %d, "msg": "%s"}`, statusCode, message)))
}
