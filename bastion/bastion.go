package bastion

import (
	"bufio"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	gliderssh "github.com/gliderlabs/ssh"
	sshclient "golang.org/x/crypto/ssh"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Init() {
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
	store := NewHostStore()

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
			printHosts(s, store.List())
			fmt.Fprint(s, "\n请输入要连接的主机 ID 或 IP（回车返回菜单）: ")
			sel, _ := readLineEcho(reader, s)
			sel = strings.TrimSpace(sel)
			if sel != "" {
				connectHostFlow(s, store, sel, reader)
			}
		case "R":
			store.Refresh()
			fmt.Fprintln(s, " 主机列表已刷新。")
		case "EXIT":
			fmt.Fprintln(s, "再见！")
			return
		default:
			// 尝试按ID或IP连接
			connectHostFlow(s, store, cmd, reader)
		}

		// 从远端退出后，或命令处理完毕，重新显示欢迎信息和菜单
		printWelcome(s, s.User())
	}
}

func printWelcome(s gliderssh.Session, user string) {
	fmt.Fprintf(s, "\n%s, 欢迎使用 OPS-GO 云堡垒机\n", user)
	fmt.Fprintln(s, "----------------------------------------------")
	fmt.Fprintln(s, " - 输入 L 查询主机列表")
	fmt.Fprintln(s, " - 输入 R 刷新主机列表")
	fmt.Fprintln(s, " - 输入主机 ID, 名称或 IP 登录主机")
	fmt.Fprintln(s, " - 输入 H 显示帮助信息")
	fmt.Fprintln(s, " - 输入 exit 退出堡垒机")
	fmt.Fprintln(s, "----------------------------------------------")
}

func printHosts(s gliderssh.Session, hosts []Host) {
	fmt.Fprintln(s, "\nID   主机名称        规格    状态       IP")
	fmt.Fprintln(s, "----------------------------------------------")
	for _, h := range hosts {
		fmt.Fprintf(s, "%-4d %-14s %-7s %-10s %s\n", h.ID, h.Name, h.Spec, h.Status, h.IP)
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
	fmt.Fprintf(s, "\n将连接主机 %s (%s)\n", host.Name, host.IP)

	username := promptEcho(reader, s, "请输入远程主机用户名: ")
	password := promptSilent(reader, s, "\n请输入远程主机密码: ")

	if err := proxyToRemote(s, host.IP, username, password); err != nil {
		fmt.Fprintf(s, "\n连接失败：%v\n", err)
		return
	}
	fmt.Fprintln(s, "\n已从远程主机退出，返回堡垒机。")
}

func proxyToRemote(s gliderssh.Session, ip, user, pass string) error {
	addr := net.JoinHostPort(ip, "22")
	cfg := &sshclient.ClientConfig{
		User:            user,
		Auth:            []sshclient.AuthMethod{sshclient.Password(pass)},
		HostKeyCallback: sshclient.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	conn, err := sshclient.Dial("tcp", addr, cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	cs, err := conn.NewSession()
	if err != nil {
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
		width := pty.Window.Width
		height := pty.Window.Height
		// 兜底尺寸，防止初始宽高为0导致过度换行
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
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
			}
		}()
	}

	cs.Stdout = s
	cs.Stderr = s
	cs.Stdin = s

	// 启动远端 shell 并阻塞到退出
	if err := cs.Shell(); err != nil {
		return err
	}
	return cs.Wait()
}

// --------- 简单的主机模型与存储 ---------
type Host struct {
	ID     int
	Name   string
	Spec   string
	Status string
	IP     string
}

type HostStore struct {
	hosts []Host
}

func NewHostStore() *HostStore {
	return &HostStore{hosts: defaultHosts()}
}

func (h *HostStore) Refresh() {
	for i := range h.hosts {
		switch rand.Intn(3) {
		case 0:
			h.hosts[i].Status = "running"
		case 1:
			h.hosts[i].Status = "stopped"
		default:
			h.hosts[i].Status = "unknown"
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

func defaultHosts() []Host {
	return []Host{
		{ID: 1, Name: "web-01", Spec: "2C4G", Status: "running", IP: "47.101.39.88"},
		{ID: 2, Name: "db-01", Spec: "4C8G", Status: "stopped", IP: "192.168.1.102"},
		{ID: 3, Name: "cache-01", Spec: "2C2G", Status: "unknown", IP: "192.168.1.103"},
	}
}

// --------- 自定义认证（示例） ---------
var allowedUsers = map[string]string{
	"admin": "admin",
	"ops":   "ops",
}

func passwordAuth(ctx gliderssh.Context, pass string) bool {
	user := ctx.User()
	if p, ok := allowedUsers[user]; ok {
		return p == pass
	}
	// 未配置用户时拒绝
	return false
}
