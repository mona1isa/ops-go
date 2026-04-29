package instance

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// WebSocketTerminator WebSocket 会话终止器
type WebSocketTerminator struct{}

// Terminate 实现 instance.SessionTerminator 接口
func (t *WebSocketTerminator) Terminate(sessionID string) error {
	controller := &WebSocketController{}
	return controller.TerminateSession(sessionID)
}

// 全局 WebSocket 终止器
var webSocketTerminator = &WebSocketTerminator{}

// GetWebSocketTerminator 获取 WebSocket 终止器实例
func GetWebSocketTerminator() *WebSocketTerminator {
	return webSocketTerminator
}

type WebSocketController struct {
	controllers.BaseController
}

// SSHSession 管理SSH会话
type SSHSession struct {
	Conn          *ssh.Client
	Session       *ssh.Session
	StdoutPipe    io.Reader
	StdinPipe     io.WriteCloser
	InstanceId    int
	UserId        int
	KeyId         int
	Recorder      *utils.SessionRecorder // 会话录制器
	SessionRecord *models.OpsSessionRecord // 会话记录
	WebSocketConn *websocket.Conn        // WebSocket 连接引用（用于发送终止消息）
	InputBuffer   string                 // 用户输入缓冲区（用于高危指令拦截）
	IsAdmin       bool                   // 是否为管理员
	mu            sync.Mutex
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

	isAdmin := userId == controllers.AdminUserId

	// 如果只有一个凭证，直接连接；否则返回凭证列表
	if len(keys) == 1 {
		// 直接连接
		if err := c.connectToInstance(conn, sessionID, userId, instanceId, keys[0], isAdmin); err != nil {
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
			if err := c.connectToInstance(conn, sessionID, userId, instanceId, *selectedKey, isAdmin); err != nil {
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
func (c *WebSocketController) connectToInstance(conn *websocket.Conn, sessionID string, userId, instanceId int, key models.OpsKey, isAdmin bool) error {
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

	// 创建会话记录
	sessionRecord := &models.OpsSessionRecord{
		SessionID:    sessionID,
		UserID:       userId,
		InstanceID:   instanceId,
		InstanceName: instance.Name,
		InstanceIP:   instance.Ip,
		KeyID:        int(key.ID),
		KeyName:      key.Name,
		KeyUser:      key.User,
		StartTime:    time.Now(),
		Status:       models.SessionStatusActive,
	}
	if err := models.DB.Create(sessionRecord).Error; err != nil {
		log.Printf("创建会话记录失败: %v", err)
	}

	// 获取明文凭证（密码和密钥都可能加密存储）
	credentials, err := utils.DecryptKey(key.Credentials)
	if err != nil {
		log.Printf("解密凭证失败: %v", err)
		return errors.New("解密凭证失败")
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

	// 请求伪终端（默认大小 80x24）
	rows := 24
	cols := 80
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		session.Close()
		sshConn.Close()
		return errors.New("请求伪终端失败: " + err.Error())
	}

	// 创建会话录制器（存储路径从配置读取，这里先硬编码）
	recorder, err := utils.NewSessionRecorder(sessionID, cols, rows, "./recordings")
	if err != nil {
		log.Printf("创建会话录制器失败: %v", err)
		// 录制失败不影响连接，继续执行
	}

	// 保存会话
	sshSession := &SSHSession{
		Conn:          sshConn,
		Session:       session,
		StdoutPipe:    stdoutPipe,
		StdinPipe:     stdinPipe,
		InstanceId:    instanceId,
		UserId:        userId,
		KeyId:         int(key.ID),
		Recorder:      recorder,
		SessionRecord: sessionRecord,
		WebSocketConn: conn, // 保存 WebSocket 连接引用
		IsAdmin:       isAdmin,
	}

	sessionMutex.Lock()
	activeSessions[sessionID] = sshSession
	sessionMutex.Unlock()

	// 启动SSH会话
	if err := session.Shell(); err != nil {
		session.Close()
		sshConn.Close()
		if recorder != nil {
			recorder.Close()
		}
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
			data := string(buf[:n])
			
			// 录制输出数据
			if sshSession.Recorder != nil {
				if err := sshSession.Recorder.RecordOutput(data); err != nil {
					log.Printf("录制输出数据失败: %v", err)
				}
			}

			// 发送到 WebSocket
			if err := conn.WriteJSON(WebSocketMessage{
				Type: "data",
				Data: data,
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

	// 更新录制器的终端大小
	if sshSession.Recorder != nil {
		sshSession.Recorder.Resize(msg.Cols, msg.Rows)
	}

	// 更新 SSH 会话的终端大小
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

	data := msg.Data

	// 如果数据包含 Escape 序列（方向键、Home、End 等），不进入buffer
	if strings.Contains(data, "\x1b") {
		if _, err := sshSession.StdinPipe.Write([]byte(data)); err != nil {
			log.Printf("写入SSH输入失败: %v", err)
		}
		return
	}

	// 逐个字符处理
	for i := 0; i < len(data); i++ {
		ch := data[i]
		switch ch {
		case '\r', '\n':
			// 回车/换行，提取完整命令并检查
			cmd := strings.TrimSpace(sshSession.InputBuffer)
			sshSession.InputBuffer = ""
			if cmd == "" {
				if _, err := sshSession.StdinPipe.Write([]byte{ch}); err != nil {
					log.Printf("写入SSH输入失败: %v", err)
				}
				continue
			}
			// 检查高危指令
			blocked, ruleName, description := instance.CheckCommand(cmd, sshSession.IsAdmin)
			if blocked {
				// 发送 Ctrl+C 取消当前输入
				if _, err := sshSession.StdinPipe.Write([]byte{0x03}); err != nil {
					log.Printf("写入SSH输入失败: %v", err)
				}
				// 发送警告到前端
				warning := fmt.Sprintf("\r\n\x1b[31m[高危指令拦截] 命令 \"%s\" 已被系统阻止执行（规则：%s）\x1b[0m\r\n", cmd, ruleName)
				if description != "" {
					warning = fmt.Sprintf("\r\n\x1b[31m[高危指令拦截] 命令 \"%s\" 已被系统阻止执行（%s）\x1b[0m\r\n", cmd, description)
				}
				if err := sshSession.WebSocketConn.WriteJSON(WebSocketMessage{
					Type: "blocked",
					Data: warning,
				}); err != nil {
					log.Printf("发送拦截警告失败: %v", err)
				}
			} else {
				if _, err := sshSession.StdinPipe.Write([]byte{ch}); err != nil {
					log.Printf("写入SSH输入失败: %v", err)
				}
			}
		case '\b', 0x7f:
			// 退格键
			if len(sshSession.InputBuffer) > 0 {
				sshSession.InputBuffer = sshSession.InputBuffer[:len(sshSession.InputBuffer)-1]
			}
			if _, err := sshSession.StdinPipe.Write([]byte{ch}); err != nil {
				log.Printf("写入SSH输入失败: %v", err)
			}
		case 0x03:
			// Ctrl+C，清空buffer
			sshSession.InputBuffer = ""
			if _, err := sshSession.StdinPipe.Write([]byte{ch}); err != nil {
				log.Printf("写入SSH输入失败: %v", err)
			}
		case 0x04:
			// Ctrl+D
			sshSession.InputBuffer = ""
			if _, err := sshSession.StdinPipe.Write([]byte{ch}); err != nil {
				log.Printf("写入SSH输入失败: %v", err)
			}
		case 0x15:
			// Ctrl+U，清空当前行
			sshSession.InputBuffer = ""
			if _, err := sshSession.StdinPipe.Write([]byte{ch}); err != nil {
				log.Printf("写入SSH输入失败: %v", err)
			}
		case '\t':
			// Tab，忽略不进入buffer
			if _, err := sshSession.StdinPipe.Write([]byte{ch}); err != nil {
				log.Printf("写入SSH输入失败: %v", err)
			}
		default:
			// 普通字符，追加到buffer并发送给SSH
			sshSession.InputBuffer += string(ch)
			if _, err := sshSession.StdinPipe.Write([]byte{ch}); err != nil {
				log.Printf("写入SSH输入失败: %v", err)
			}
		}
	}
}

// handleClose 关闭SSH会话
func (c *WebSocketController) handleClose(sessionID string) {
	c.closeSession(sessionID, models.SessionStatusCompleted)
}

// TerminateSession 终止会话（管理员强制终止）
func (c *WebSocketController) TerminateSession(sessionID string) error {
	sessionMutex.RLock()
	sshSession, ok := activeSessions[sessionID]
	sessionMutex.RUnlock()

	if !ok {
		return errors.New("会话不存在或已结束")
	}

	sshSession.mu.Lock()
	defer sshSession.mu.Unlock()

	// 向用户发送终止消息
	if sshSession.WebSocketConn != nil {
		message := "\r\n\r\n========================================\r\n"
		message += "⚠️  警告：会话已被管理员终止\r\n"
		message += "========================================\r\n"
		message += "原因：管理员强制结束该会话\r\n"
		message += "时间：" + time.Now().Format("2006-01-02 15:04:05") + "\r\n"
		message += "========================================\r\n"
		message += "\r\n无法继续执行远程操作，连接即将关闭...\r\n\r\n"

		// 发送终止消息到 WebSocket
		if err := sshSession.WebSocketConn.WriteJSON(WebSocketMessage{
			Type: "terminated",
			Data: message,
		}); err != nil {
			log.Printf("发送终止消息失败: %v", err)
		}

		// 给用户一点时间看到消息
		time.Sleep(500 * time.Millisecond)
	}

	// 更新会话记录状态为已终止
	now := time.Now()
	if sshSession.SessionRecord != nil {
		duration := int(now.Sub(sshSession.SessionRecord.StartTime).Seconds())
		var fileSize int64
		if sshSession.Recorder != nil {
			fileSize, _ = sshSession.Recorder.GetFileSize()
		}
		models.DB.Model(&models.OpsSessionRecord{}).
			Where("session_id = ?", sessionID).
			Updates(map[string]interface{}{
				"end_time":  &now,
				"duration":  duration,
				"status":    models.SessionStatusAborted,
				"file_size": fileSize,
			})
	}

	// 关闭标准输入管道（阻止用户继续输入）
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

	// 关闭录制器
	if sshSession.Recorder != nil {
		if err := sshSession.Recorder.Close(); err != nil {
			log.Printf("关闭录制器失败: %v", err)
		}
	}

	// 关闭 WebSocket 连接
	if sshSession.WebSocketConn != nil {
		// 先发送关闭消息
		_ = sshSession.WebSocketConn.WriteJSON(WebSocketMessage{
			Type: "close",
			Data: "会话已终止",
		})
		// 延迟一下，让前端收到消息
		time.Sleep(100 * time.Millisecond)
		// 关闭连接
		_ = sshSession.WebSocketConn.Close()
	}

	// 从活跃会话列表中移除
	sessionMutex.Lock()
	delete(activeSessions, sessionID)
	sessionMutex.Unlock()

	log.Printf("Web Terminal 会话已终止: %s", sessionID)
	return nil
}

// closeSession 关闭SSH会话（内部方法）
func (c *WebSocketController) closeSession(sessionID string, status int8) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	if sshSession, ok := activeSessions[sessionID]; ok {
		sshSession.mu.Lock()
		defer sshSession.mu.Unlock()

		// 关闭录制器
		if sshSession.Recorder != nil {
			if err := sshSession.Recorder.Close(); err != nil {
				log.Printf("关闭录制器失败: %v", err)
			}
			// 更新会话记录
			if sshSession.SessionRecord != nil {
				fileSize, _ := sshSession.Recorder.GetFileSize()
				duration := sshSession.Recorder.GetDuration()
				endTime := time.Now()
				updates := map[string]interface{}{
					"end_time":       &endTime,
					"duration":       duration,
					"status":         status,
					"recording_file": sshSession.Recorder.GetFilePath(),
					"file_size":      fileSize,
				}
				models.DB.Model(&models.OpsSessionRecord{}).
					Where("session_id = ?", sessionID).
					Updates(updates)
			}
		}

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
