package bastion

import (
	"bufio"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	gliderssh "github.com/gliderlabs/ssh"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/instance"
	"github.com/zhany/ops-go/utils"
	"golang.org/x/crypto/bcrypt"
	sshclient "golang.org/x/crypto/ssh"
)

// SessionManager 会话管理器
type SessionManager struct {
	sessions map[string]*ActiveSession
	mu       sync.RWMutex
}

// ActiveSession 活跃会话
type ActiveSession struct {
	SessionID   string
	UserID      int
	UserName    string
	InstanceID  int
	InstanceIP  string
	Conn        *sshclient.Client
	Session     *sshclient.Session
	SSHSession  gliderssh.Session
	StartTime   time.Time
}

var globalSessionManager = &SessionManager{
	sessions: make(map[string]*ActiveSession),
}

// GetGlobalSessionManager 获取全局会话管理器（导出给外部使用）
func GetGlobalSessionManager() *SessionManager {
	return globalSessionManager
}

// GetSessionManager 获取全局会话管理器
func GetSessionManager() *SessionManager {
	return globalSessionManager
}

// AddSession 添加活跃会话
func (sm *SessionManager) AddSession(sessionID string, session *ActiveSession) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[sessionID] = session
	log.Printf("会话已添加到管理器: %s", sessionID)
}

// RemoveSession 移除会话
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
	log.Printf("会话已从管理器移除: %s", sessionID)
}

// GetSession 获取会话
func (sm *SessionManager) GetSession(sessionID string) *ActiveSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[sessionID]
}

// ListSessions 列出所有活跃会话
func (sm *SessionManager) ListSessions() []*ActiveSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sessions := make([]*ActiveSession, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// TerminateSession 终止会话
func (sm *SessionManager) TerminateSession(sessionID string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("会话不存在")
	}

	// 向用户发送终止消息
	if session.SSHSession != nil {
		message := "\r\n\r\n========================================\r\n"
		message += "⚠️  警告：会话已被管理员终止\r\n"
		message += "========================================\r\n"
		message += "原因：管理员强制结束该会话\r\n"
		message += "时间：" + time.Now().Format("2006-01-02 15:04:05") + "\r\n"
		message += "========================================\r\n"
		message += "\r\n无法继续执行远程操作，连接即将关闭...\r\n\r\n"
		fmt.Fprintf(session.SSHSession, "%s", message)
		// 给用户一点时间看到消息
		time.Sleep(500 * time.Millisecond)
	}

	// 先关闭 SSH 会话，停止远端命令执行
	if session.Session != nil {
		_ = session.Session.Close()
	}

	// 关闭 SSH 连接
	if session.Conn != nil {
		_ = session.Conn.Close()
	}

	// 关闭 gliderssh 会话，断开客户端连接
	if session.SSHSession != nil {
		_ = session.SSHSession.Close()
	}

	// 更新数据库记录状态
	now := time.Now()
	duration := int(now.Sub(session.StartTime).Seconds())

	// 获取当前文件大小
	var fileSize int64
	if record, err := (&instance.SessionRecordService{}).GetBySessionID(sessionID); err == nil && record.RecordingFile != "" {
		if info, err := os.Stat(record.RecordingFile); err == nil {
			fileSize = info.Size()
		}
	}

	models.DB.Model(&models.OpsSessionRecord{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"end_time":       &now,
			"duration":       duration,
			"status":         models.SessionStatusAborted,
			"recording_file": fmt.Sprintf("recordings/%s/%s.cast", now.Format("2006-01-02"), sessionID),
			"file_size":      fileSize,
		})

	log.Printf("会话已终止: %s", sessionID)

	return nil
}

// Terminate 实现 instance.SessionTerminator 接口
func (sm *SessionManager) Terminate(sessionID string) error {
	return sm.TerminateSession(sessionID)
}

func Init() {
	// 注册会话终止器，让 Service 层可以调用 Bastion 的终止功能
	instance.RegisterTerminator(globalSessionManager)

	pem := loadOrCreateHostKeyPEM(filepath.Join("cmd", "bastion", "hostkey.pem"))

	gliderssh.Handle(sessionHandler)

	log.Printf("Bastion SSH server listening on :2222")
	log.Fatal(gliderssh.ListenAndServe(
		":2222",
		nil,
		gliderssh.HostKeyPEM(pem),
		gliderssh.PasswordAuth(passwordAuth),
	))
}

// --------- 会话处理与交互菜单 ---------
func sessionHandler(s gliderssh.Session) {
	cmd := s.Command()
	// 如果是交互式登录（没有附带命令），进入堡垒机菜单
	if len(cmd) == 0 {
		interactiveBastion(s)
		return
	}
	// 如果用户通过 ssh user@host "command" 方式调用，这里简单响应
	switch strings.ToLower(cmd[0]) {
	case "date":
		fmt.Fprintf(s, "当前时间：%s\n", time.Now().Format(time.RFC3339))
	case "whoami":
		fmt.Fprintf(s, "当前用户：%s\n", s.User())
	default:
		fmt.Fprintf(s, "未知命令：%s\n", cmd[0])
	}
}

// 持久化加载或生成主机密钥
func loadOrCreateHostKeyPEM(path string) []byte {
	// 确保目录存在
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	b, err := os.ReadFile(path)
	if err == nil && len(b) > 0 {
		return b
	}
	// 不存在则生成并保存
	b = generateHostKeyPEM()
	_ = os.WriteFile(path, b, 0600)
	return b
}

// --------- HostKey 生成 ---------
func generateHostKeyPEM() []byte {
	key, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		log.Fatalf("generate host key failed: %v", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER}
	return pem.EncodeToMemory(block)
}

func interactiveBastion(s gliderssh.Session) {
	reader := bufio.NewReader(s)
	store := NewHostStore(s.User())

	printWelcome(s, s.User())

	for {
		fmt.Fprint(s, "\nConsole> ")
		line, err := readLineEcho(reader, s)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(s, "\n读取输入出错: %v\n", err)
			return
		}
		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}

		switch strings.ToUpper(cmd) {
		case "L":
			// 显示主机列表（不重新查询数据库）
			printHosts(s, store.List())
		case "R":
			// 刷新主机列表（重新查询数据库）
			store.Refresh()
			fmt.Fprintln(s, "\n主机列表已刷新。")
			printHosts(s, store.List())
		case "C":
			// 清除屏幕
			fmt.Fprint(s, "\033[2J\033[H")
		case "H":
			// 显示菜单帮助信息
			printWelcome(s, s.User())
		case "EXIT":
			fmt.Fprintln(s, "再见！")
			return
		default:
			// 尝试按ID或IP连接
			connectHostFlow(s, store, cmd, reader)
		}

		// 从远端退出后，或命令处理完毕，重新显示欢迎信息和菜单
		// printWelcome(s, s.User())
	}
}

// getDisplayWidth 计算字符串在终端中的显示宽度（中文字符算2个字符宽度）
func getDisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			// 中文字符范围
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// padRightRight 将字符串右侧填充到指定显示宽度（考虑中文字符宽度）
func padRight(s string, width int) string {
	displayWidth := getDisplayWidth(s)
	if displayWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-displayWidth)
}

// padLeft 将字符串左侧填充到指定显示宽度（考虑中文字符宽度）
func padLeft(s string, width int) string {
	displayWidth := getDisplayWidth(s)
	if displayWidth >= width {
		return s
	}
	return strings.Repeat(" ", width-displayWidth) + s
}

func printWelcome(s gliderssh.Session, user string) {
	fmt.Fprintf(s, "\n%s, 欢迎使用 OPS-GO 云堡垒机\n", user)
	fmt.Fprintln(s, "----------------------------------------------")
	fmt.Fprintln(s, " - 输入 L 查询主机列表")
	fmt.Fprintln(s, " - 输入 R 刷新主机列表")
	fmt.Fprintln(s, " - 输入主机 ID, 名称或 IP 登录主机")
	fmt.Fprintln(s, " - 输入 H 显示帮助信息")
	fmt.Fprintln(s, " - 输入 C 清除当前屏幕")
	fmt.Fprintln(s, " - 输入 exit 退出堡垒机")
	fmt.Fprintln(s, "----------------------------------------------")
}

func printHosts(s gliderssh.Session, hosts []Host) {
	// 列宽设置（显示宽度）
	idWidth := 5
	nameWidth := 19
	specWidth := 11
	statusWidth := 10
	ipWidth := 18

	// 打印表头
	fmt.Fprintln(s, "")
	fmt.Fprintf(s, " %s | %s | %s | %s | %s\n",
		padRight("ID", idWidth),
		padRight("主机名称", nameWidth),
		padRight("规格", specWidth),
		padRight("状态", statusWidth),
		padRight("IP地址", ipWidth))
	fmt.Fprintln(s, " "+strings.Repeat("-", idWidth+1)+"+"+strings.Repeat("-", nameWidth+2)+"+"+
		strings.Repeat("-", specWidth+2)+"+"+strings.Repeat("-", statusWidth+2)+"+"+
		strings.Repeat("-", ipWidth+1))

	// 打印主机列表
	for _, h := range hosts {
		fmt.Fprintf(s, " %s | %s | %s | %s | %s\n",
			padRight(fmt.Sprintf("%d", h.ID), idWidth),
			padRight(h.Name, nameWidth),
			padRight(h.Spec, specWidth),
			padRight(h.Status, statusWidth),
			padRight(h.IP, ipWidth))
	}
}

func readLine(r *bufio.Reader) (string, error) {
	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			if len(buf) > 0 {
				return strings.TrimRight(string(buf), "\r\n"), nil
			}
			return "", err
		}
		// 当检测到回车或换行时返回（兼容仅发送'\r'的客户端）
		if b == '\n' || b == '\r' {
			return strings.TrimRight(string(buf), "\r\n"), nil
		}
		buf = append(buf, b)
	}
}

func readLineEcho(r *bufio.Reader, w io.Writer) (string, error) {
	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			if len(buf) > 0 {
				return strings.TrimRight(string(buf), "\r\n"), nil
			}
			return "", err
		}
		// 处理回车或换行
		if b == '\n' || b == '\r' {
			return strings.TrimRight(string(buf), "\r\n"), nil
		}
		// 处理退格键（8 或 127）
		if b == 8 || b == 127 {
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				// 回显删除效果
				fmt.Fprint(w, "\b \b")
			}
			continue
		}
		// 简单可打印字符回显
		if b >= 32 && b != 127 {
			buf = append(buf, b)
			_, _ = w.Write([]byte{b})
		}
	}
}

func promptEcho(r *bufio.Reader, s gliderssh.Session, msg string) string {
	fmt.Fprint(s, msg)
	text, _ := readLineEcho(r, s)
	return strings.TrimSpace(text)
}

func promptSilent(r *bufio.Reader, s gliderssh.Session, msg string) string {
	fmt.Fprint(s, msg)
	text, _ := readLine(r)
	return strings.TrimSpace(text)
}

func connectHostFlow(s gliderssh.Session, store *HostStore, sel string, reader *bufio.Reader) {
	var host *Host
	if id, err := strconv.Atoi(sel); err == nil {
		host = store.FindByID(id)
	} else {
		host = store.FindByIP(sel)
	}
	if host == nil {
		host = store.FindByName(sel)
	}

	if host == nil {
		fmt.Fprintf(s, "未找到主机：%s\n", sel)
		return
	}

	// 检查主机状态
	if host.Status != "running" {
		fmt.Fprintf(s, "主机 %s (%s) 当前状态：%s\n", host.Name, host.IP, host.Status)
		fmt.Fprintln(s, "主机未处于运行状态，无法访问")
		return
	}

	fmt.Fprintf(s, "\n将连接主机 %s (%s)\n", host.Name, host.IP)

	// 获取用户有权限的密钥
	keyAuthService := &instance.UserInstanceKeyAuth{
		UserId:     store.userID,
		InstanceId: host.ID,
		AuthType:   1,
	}
	keys, err := keyAuthService.GetUserInstanceKeyAuth()
	if err != nil {
		fmt.Fprintf(s, "获取用户密钥失败：%v\n", err)
		return
	}

	if len(keys) == 0 {
		fmt.Fprintln(s, "您没有该主机的登录凭证权限")
		return
	}

	var selectedKey models.OpsKey

	if len(keys) == 1 {
		selectedKey = keys[0]
		fmt.Fprintf(s, "\n检测到唯一凭证，自动选择：%s (%s)\n", selectedKey.Name, selectedKey.User)
	} else {
		// 显示密钥列表
		fmt.Fprintln(s, "\n请选择登录凭证：")
		for i, key := range keys {
			fmt.Fprintf(s, "%d. %s (%s) - %s\n", i+1, key.Name, key.User, key.Protocol)
		}

		// 让用户选择密钥
		keyIndexStr := promptEcho(reader, s, "\n请输入凭证序号：")
		keyIndex, err := strconv.Atoi(keyIndexStr)
		if err != nil || keyIndex < 1 || keyIndex > len(keys) {
			fmt.Fprintln(s, "无效的凭证序号")
			return
		}
		selectedKey = keys[keyIndex-1]
	}

	log.Printf("selectedKey: %+v", selectedKey)
	// 获取明文凭证（密码和密钥都可能加密存储）
	credentials, err := utils.DecryptKey(selectedKey.Credentials)
	if err != nil {
		log.Printf("解密凭证失败: %v", err)
		fmt.Fprintln(s, "解密凭证失败")
		return
	}

	// 判断是否为管理员
	isAdmin := store.userID == models.AdminUserId || store.user == "admin"

	// 连接到远程主机
	if err := proxyToRemote(s, host, selectedKey, credentials, keyAuthService.UserId, isAdmin); err != nil {
		fmt.Fprintf(s, "\n连接失败：%v\n", err)
		return
	}
	fmt.Fprintln(s, "\n已从远程主机退出，返回堡垒机。")
}

func proxyToRemote(s gliderssh.Session, host *Host, key models.OpsKey, credentials string, userId int, isAdmin bool) error {
	// 生成会话ID
	sessionID := generateSessionID(s.User(), host.ID)

	// 创建会话记录
	record := &models.OpsSessionRecord{
		SessionID:     sessionID,
		UserID:        userId,
		InstanceID:    host.ID,
		InstanceName:  host.Name,
		InstanceIP:    host.IP,
		KeyID:         key.ID,
		KeyName:       key.Name,
		KeyUser:       key.User,
		StartTime:     time.Now(),
		Status:        models.SessionStatusActive,
	}
	if err := models.DB.Create(record).Error; err != nil {
		log.Printf("创建会话记录失败: %v", err)
		// 即使创建记录失败，也继续连接
	}

	// 获取终端尺寸
	var width, height int
	if pty, _, ok := s.Pty(); ok {
		width = pty.Window.Width
		height = pty.Window.Height
		// 兜底尺寸
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
		}
	}

	// 创建会话录制器
	recorder, err := utils.NewSessionRecorder(sessionID, width, height, "recordings")
	if err != nil {
		log.Printf("创建录制器失败: %v", err)
		// 即使录制失败，也继续连接
		recorder = nil
	}
	if recorder != nil {
		defer recorder.Close()
	}

	addr := net.JoinHostPort(host.IP, "22")

	// 构建认证方法
	var authMethods []sshclient.AuthMethod

	// 尝试将凭证解析为密钥，失败则作为密码处理
	sshKey, err := sshclient.ParsePrivateKey([]byte(credentials))
	if err == nil {
		// 密钥认证
		authMethods = append(authMethods, sshclient.PublicKeys(sshKey))
	} else {
		// 密码认证
		authMethods = append(authMethods, sshclient.Password(credentials))
	}

	cfg := &sshclient.ClientConfig{
		User:            key.User,
		Auth:            authMethods,
		HostKeyCallback: sshclient.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	conn, err := sshclient.Dial("tcp", addr, cfg)
	if err != nil {
		// 更新会话记录状态为异常中断
		updateSessionRecordStatus(sessionID, models.SessionStatusAborted, 0)
		return err
	}
	defer conn.Close()

	cs, err := conn.NewSession()
	if err != nil {
		// 更新会话记录状态为异常中断
		updateSessionRecordStatus(sessionID, models.SessionStatusAborted, 0)
		return err
	}
	defer cs.Close()

	if pty, winCh, ok := s.Pty(); ok {
		modes := sshclient.TerminalModes{
			sshclient.ECHO:          1,
			sshclient.TTY_OP_ISPEED: 14400,
			sshclient.TTY_OP_OSPEED: 14400,
		}
		term := pty.Term
		if term == "" {
			term = "xterm-256color"
		}

		// 远端请求与本地相同大小的终端（注意参数顺序：height, width）
		_ = cs.RequestPty(term, height, width, modes)
		// 监听窗口大小变化并同步到远端会话
		go func() {
			for win := range winCh {
				w := win.Width
				h := win.Height
				if w <= 0 {
					w = 80
				}
				if h <= 0 {
					h = 24
				}
				// 注意参数顺序：height, width
				_ = cs.WindowChange(h, w)
				// 同时更新录制器的终端尺寸
				if recorder != nil {
					recorder.Resize(w, h)
				}
			}
		}()
	}

	// 使用 StdinPipe 启动输入处理 goroutine（包含高危指令拦截）
	stdinPipe, err := cs.StdinPipe()
	if err != nil {
		// 更新会话记录状态为异常中断
		updateSessionRecordStatus(sessionID, models.SessionStatusAborted, 0)
		return err
	}
	go handleBastionInput(s, stdinPipe, recorder, isAdmin)

	// 设置输出
	stdout := newRecordingWriter(s, recorder, true)
	stderr := newRecordingWriter(s, recorder, true)
	cs.Stdout = stdout
	cs.Stderr = stderr

	// 启动远端 shell 并阻塞到退出
	if err := cs.Shell(); err != nil {
		// 更新会话记录状态为异常中断
		updateSessionRecordStatus(sessionID, models.SessionStatusAborted, 0)
		return err
	}

	// 添加到会话管理器
	activeSession := &ActiveSession{
		SessionID:  sessionID,
		UserID:     userId,
		UserName:   s.User(),
		InstanceID: host.ID,
		InstanceIP: host.IP,
		Conn:       conn,
		Session:    cs,
		SSHSession: s,
		StartTime:  record.StartTime,
	}
	globalSessionManager.AddSession(sessionID, activeSession)
	defer globalSessionManager.RemoveSession(sessionID)

	err = cs.Wait()

	// 会话结束，更新记录
	endTime := time.Now()
	duration := int(endTime.Sub(record.StartTime).Seconds())

	status := models.SessionStatusCompleted
	if err != nil {
		status = models.SessionStatusAborted
	}

	var fileSize int64
	if recorder != nil {
		if size, err := recorder.GetFileSize(); err == nil {
			fileSize = size
		}
	}

	updateSessionRecord(sessionID, endTime, duration, status, fileSize)

	return err
}

// generateSessionID 生成会话ID
func generateSessionID(user string, instanceID int) string {
	return fmt.Sprintf("%s-%d-%d", user, instanceID, time.Now().UnixNano())
}

// updateSessionRecordStatus 更新会话记录状态
func updateSessionRecordStatus(sessionID string, status int8, duration int) {
	now := time.Now()
	models.DB.Model(&models.OpsSessionRecord{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"status":   status,
			"end_time": &now,
			"duration": duration,
		})
}

// updateSessionRecord 更新会话记录
func updateSessionRecord(sessionID string, endTime time.Time, duration int, status int8, fileSize int64) {
	models.DB.Model(&models.OpsSessionRecord{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"end_time":      &endTime,
			"duration":       duration,
			"status":        status,
			"recording_file": fmt.Sprintf("recordings/%s/%s.cast", endTime.Format("2006-01-02"), sessionID),
			"file_size":     fileSize,
		})
}

// recordingReader 包装读取器，记录输入数据
type recordingReader struct {
	reader   io.Reader
	recorder *utils.SessionRecorder
	record   bool
}

func newRecordingReader(reader io.Reader, recorder *utils.SessionRecorder, record bool) *recordingReader {
	return &recordingReader{
		reader:   reader,
		recorder: recorder,
		record:   record,
	}
}

func (r *recordingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.record && r.recorder != nil {
		_ = r.recorder.RecordInput(string(p[:n]))
	}
	return n, err
}

// recordingWriter 包装写入器，记录输出数据
type recordingWriter struct {
	writer   io.Writer
	recorder *utils.SessionRecorder
	record   bool
}

func newRecordingWriter(writer io.Writer, recorder *utils.SessionRecorder, record bool) *recordingWriter {
	return &recordingWriter{
		writer:   writer,
		recorder: recorder,
		record:   record,
	}
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	if w.record && w.recorder != nil {
		_ = w.recorder.RecordOutput(string(p))
	}
	return w.writer.Write(p)
}

// handleBastionInput 处理堡垒机输入，包含高危指令拦截
func handleBastionInput(s gliderssh.Session, stdinPipe io.WriteCloser, recorder *utils.SessionRecorder, isAdmin bool) {
	defer stdinPipe.Close()

	var inputBuffer string
	buf := make([]byte, 4096)

	for {
		n, err := s.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("读取用户输入失败: %v", err)
			}
			return
		}
		if n <= 0 {
			continue
		}

		data := string(buf[:n])

		// 录制输入数据
		if recorder != nil {
			if err := recorder.RecordInput(data); err != nil {
				log.Printf("录制输入数据失败: %v", err)
			}
		}

		// 如果数据包含 Escape 序列（方向键、Home、End 等），不进入buffer
		if strings.Contains(data, "\x1b") {
			if _, err := stdinPipe.Write([]byte(data)); err != nil {
				log.Printf("写入SSH输入失败: %v", err)
				return
			}
			continue
		}

		// 逐个字符处理
		for i := 0; i < len(data); i++ {
			ch := data[i]
			switch ch {
			case '\r', '\n':
				// 回车/换行，提取完整命令并检查
				cmd := strings.TrimSpace(inputBuffer)
				inputBuffer = ""
				if cmd == "" {
					if _, err := stdinPipe.Write([]byte{ch}); err != nil {
						log.Printf("写入SSH输入失败: %v", err)
						return
					}
					continue
				}
				// 检查高危指令
				blocked, ruleName, description := instance.CheckCommand(cmd, isAdmin)
				if blocked {
					// 发送 Ctrl+C 取消当前输入
					if _, err := stdinPipe.Write([]byte{0x03}); err != nil {
						log.Printf("写入SSH输入失败: %v", err)
						return
					}
					// 发送警告到用户终端
					warning := fmt.Sprintf("\r\n\x1b[31m[高危指令拦截] 命令 \"%s\" 已被系统阻止执行（规则：%s）\x1b[0m\r\n", cmd, ruleName)
					if description != "" {
						warning = fmt.Sprintf("\r\n\x1b[31m[高危指令拦截] 命令 \"%s\" 已被系统阻止执行（%s）\x1b[0m\r\n", cmd, description)
					}
					if _, err := s.Write([]byte(warning)); err != nil {
						log.Printf("发送拦截警告失败: %v", err)
					}
				} else {
					if _, err := stdinPipe.Write([]byte{ch}); err != nil {
						log.Printf("写入SSH输入失败: %v", err)
						return
					}
				}
			case '\b', 0x7f:
				// 退格键
				if len(inputBuffer) > 0 {
					inputBuffer = inputBuffer[:len(inputBuffer)-1]
				}
				if _, err := stdinPipe.Write([]byte{ch}); err != nil {
					log.Printf("写入SSH输入失败: %v", err)
					return
				}
			case 0x03:
				// Ctrl+C，清空buffer
				inputBuffer = ""
				if _, err := stdinPipe.Write([]byte{ch}); err != nil {
					log.Printf("写入SSH输入失败: %v", err)
					return
				}
			case 0x04:
				// Ctrl+D
				inputBuffer = ""
				if _, err := stdinPipe.Write([]byte{ch}); err != nil {
					log.Printf("写入SSH输入失败: %v", err)
					return
				}
			case 0x15:
				// Ctrl+U，清空当前行
				inputBuffer = ""
				if _, err := stdinPipe.Write([]byte{ch}); err != nil {
					log.Printf("写入SSH输入失败: %v", err)
					return
				}
			case '\t':
				// Tab，忽略不进入buffer
				if _, err := stdinPipe.Write([]byte{ch}); err != nil {
					log.Printf("写入SSH输入失败: %v", err)
					return
				}
			default:
				// 普通字符，追加到buffer并发送给SSH
				inputBuffer += string(ch)
				if _, err := stdinPipe.Write([]byte{ch}); err != nil {
					log.Printf("写入SSH输入失败: %v", err)
					return
				}
			}
		}
	}
}

// --------- 主机模型与存储 ---------
type Host struct {
	ID     int
	Name   string
	Spec   string
	Status string
	IP     string
}

type HostStore struct {
	hosts  []Host
	userID int
	user   string
}

func NewHostStore(user string) *HostStore {
	store := &HostStore{user: user}
	store.Refresh()
	return store
}

func (h *HostStore) Refresh() {
	// 获取用户信息
	var sysUser models.SysUser
	if err := models.DB.Where("user_name = ?", h.user).First(&sysUser).Error; err != nil {
		log.Printf("获取用户信息失败: %s, 错误: %v", h.user, err)
		return
	}
	h.userID = int(sysUser.ID)

	var instances []models.OpsInstance
	var err error

	// 判断是否为超级管理员（ID=1）或 admin 用户名（兼容旧逻辑）
	if sysUser.ID == models.AdminUserId || h.user == "admin" {
		// admin 用户查询所有主机
		if err = models.DB.Where("del_flag = ?", "0").Find(&instances).Error; err != nil {
			log.Printf("获取所有主机失败: %s, 错误: %v", h.user, err)
			return
		}
	} else {
		// 普通用户查询有权限的主机和有权限的主机分组中的主机
		authService := &instance.UserInstanceAuth{UserId: h.userID}
		instances, err = authService.GetUserInstances()
		if err != nil {
			log.Printf("获取用户主机失败: %s, 错误: %v", h.user, err)
			return
		}
	}

	// 转换为Host列表
	h.hosts = make([]Host, len(instances))
	for i, instance := range instances {
		status := "unknown"
		if instance.Status == "1" {
			status = "running"
		} else {
			status = "stopped"
		}
		h.hosts[i] = Host{
			ID:     int(instance.ID),
			Name:   instance.Name,
			Spec:   instance.Spec,
			Status: status,
			IP:     instance.Ip,
		}
	}
}

func (h *HostStore) List() []Host { return h.hosts }

func (h *HostStore) FindByID(id int) *Host {
	for i := range h.hosts {
		if h.hosts[i].ID == id {
			return &h.hosts[i]
		}
	}
	return nil
}

func (h *HostStore) FindByName(name string) *Host {
	for i := range h.hosts {
		if strings.Contains(h.hosts[i].Name, name) {
			return &h.hosts[i]
		}
	}
	return nil
}

func (h *HostStore) FindByIP(ip string) *Host {
	for i := range h.hosts {
		if h.hosts[i].IP == ip {
			return &h.hosts[i]
		}
	}
	return nil
}

// --------- 自定义认证 ---------
func passwordAuth(ctx gliderssh.Context, pass string) bool {
	user := ctx.User()
	var sysUser models.SysUser
	if err := models.DB.Where("user_name = ? AND status = '1'", user).First(&sysUser).Error; err != nil {
		log.Printf("用户认证失败: %s, 错误: %v", user, err)
		return false
	}
	if !checkPassword(pass, sysUser.Password) {
		log.Printf("用户认证失败: %s, 密码错误", user)
		return false
	}
	return true
}

func checkPassword(inputPass string, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPass))
	if err != nil {
		log.Println("密码校验失败：", err)
		return false
	}
	return true
}
