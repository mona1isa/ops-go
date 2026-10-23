package instance

import (
	"log"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// AuditContext 拦截留痕需要的会话上下文
type AuditContext struct {
	SessionID    string
	Channel      int8
	UserID       int
	UserName     string
	InstanceID   int
	InstanceName string
	InstanceIP   string
	KeyUser      string
}

// maxGuardBuffer 单行缓冲上限，避免异常输入把内存吃满
const maxGuardBuffer = 4096

// LineGuard 把「逐字符累积 + 回车时校验高危规则」的终端输入逻辑收敛到一处，
// SSH 堡垒机与 Web 终端共用，避免两份实现行为漂移。
//
// 关键点：旧实现遇到含 ESC 的输入块就整块透传、不进缓冲也不校验，粘贴一段命令
// （括号粘贴模式带 \x1b[200~ 前缀，或方向键序列里夹了回车）可直接绕过拦截。
// 这里改为按序列解析：转义序列照常转发给远端（不影响方向键等编辑功能），
// 其余字节一律走同一套缓冲与校验。
type LineGuard struct {
	isAdmin bool
	ctx     AuditContext
	buf     []byte
}

func NewLineGuard(ctx AuditContext, isAdmin bool) *LineGuard {
	return &LineGuard{isAdmin: isAdmin, ctx: ctx}
}

// Feed 处理一段终端输入，返回需要写入远端 stdin 的字节（被拦截的行以 Ctrl+C 收尾）
// 以及本次命中的拦截事件，调用方负责把事件写日志/提示给用户端。
func (g *LineGuard) Feed(data string) ([]byte, []BlockHit) {
	out := make([]byte, 0, len(data)+8)
	var hits []BlockHit

	for i := 0; i < len(data); i++ {
		ch := data[i]

		if ch == 0x1b {
			n := escapeSeqLen(data, i)
			out = append(out, data[i:i+n]...)
			i += n - 1
			continue
		}

		switch ch {
		case '\r', '\n':
			cmd := strings.TrimSpace(string(g.buf))
			g.buf = g.buf[:0]
			if cmd == "" {
				out = append(out, ch)
				continue
			}
			blocked, ruleID, ruleName, desc, normalized := CheckCommandDetailed(cmd, g.isAdmin)
			if !blocked {
				out = append(out, ch)
				continue
			}
			// 只拦回车，让远端 shell 收到 Ctrl+C 丢弃这一整行
			out = append(out, 0x03)
			hits = append(hits, BlockHit{
				RuleID: ruleID, RuleName: ruleName, Description: desc,
				Command: cmd, Normalized: normalized,
			})
		case '\b', 0x7f:
			if len(g.buf) > 0 {
				g.buf = g.buf[:len(g.buf)-1]
			}
			out = append(out, ch)
		case 0x03, 0x04, 0x15:
			// Ctrl+C / Ctrl+D / Ctrl+U：当前行作废
			g.buf = g.buf[:0]
			out = append(out, ch)
		case '\t':
			// Tab 补全结果由远端回显，内容不进缓冲（与旧实现一致）
			out = append(out, ch)
		default:
			if len(g.buf) < maxGuardBuffer {
				g.buf = append(g.buf, ch)
			}
			out = append(out, ch)
		}
	}
	return out, hits
}

// Record 为一次拦截补上会话上下文后落库并写进程日志
func (g *LineGuard) Record(hit BlockHit) {
	RecordCommandBlock(hit, g.ctx)
}

// WarningText 生成发给用户端的红字提示，格式与既有实现保持一致
func (hit BlockHit) WarningText() string {
	if hit.Description != "" {
		return "\r\n\x1b[31m[高危指令拦截] 命令 \"" + hit.Command + "\" 已被系统阻止执行（" + hit.Description + "）\x1b[0m\r\n"
	}
	return "\r\n\x1b[31m[高危指令拦截] 命令 \"" + hit.Command + "\" 已被系统阻止执行（规则：" + hit.RuleName + "）\x1b[0m\r\n"
}

// escapeSeqLen 返回 s[i]（必为 ESC）起始的终端序列长度，无法识别时按 1 字节处理
func escapeSeqLen(s string, i int) int {
	if i+1 >= len(s) {
		return 1
	}
	switch s[i+1] {
	case '[': // CSI: ESC [ 参数(0x30-0x3F) 中间字节(0x20-0x2F) 结束字节(0x40-0x7E)
		j := i + 2
		for j < len(s) && s[j] >= 0x30 && s[j] <= 0x3F {
			j++
		}
		for j < len(s) && s[j] >= 0x20 && s[j] <= 0x2F {
			j++
		}
		if j < len(s) && s[j] >= 0x40 && s[j] <= 0x7E {
			return j + 1 - i
		}
		return j - i
	case ']': // OSC: 以 BEL 或 ST(ESC \) 结束
		j := i + 2
		for j < len(s) {
			if s[j] == 0x07 {
				return j + 1 - i
			}
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2 - i
			}
			j++
		}
		return j - i
	case 'P', 'X', '^', '_': // DCS/SOS/PM/APC：以 ST 结束
		j := i + 2
		for j < len(s) {
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2 - i
			}
			j++
		}
		return j - i
	case 'O': // SS3: ESC O x
		if i+2 < len(s) {
			return 3
		}
		return 2
	case '(', ')', '#', ' ': // 字符集/线序列，ESC + 选择符 + 终字符共 3 字节
		if i+2 < len(s) {
			return 3
		}
		return 2
	default:
		if s[i+1] >= 0x40 && s[i+1] <= 0x7E {
			return 2
		}
		return 1
	}
}

var (
	blockLogOnce sync.Once
	blockLogCh   chan models.OpsCommandBlockLog
)

// startBlockLogWriter 启动拦截记录的异步写入协程
func startBlockLogWriter() {
	blockLogOnce.Do(func() {
		blockLogCh = make(chan models.OpsCommandBlockLog, 256)
		go func() {
			for rec := range blockLogCh {
				if err := models.DB.Create(&rec).Error; err != nil {
					log.Printf("写入高危指令拦截记录失败: %v", err)
				}
			}
		}()
	})
}

// RecordCommandBlock 落一条拦截记录。走通道异步写，终端 I/O 不被数据库拖慢；
// 通道满时丢弃并记进程日志，宁可丢留痕也不能卡住用户会话。
func RecordCommandBlock(hit BlockHit, ctx AuditContext) {
	log.Printf("[高危指令拦截] 用户 %s(id=%d) 渠道=%d 会话=%s 主机=%s(%s) 命令=%q 命中规则=%s 形态=%q",
		ctx.UserName, ctx.UserID, ctx.Channel, ctx.SessionID, ctx.InstanceName, ctx.InstanceIP,
		hit.Command, hit.RuleName, hit.Normalized)

	if blockLogCh == nil {
		return
	}
	rec := models.OpsCommandBlockLog{
		SessionID:    ctx.SessionID,
		Channel:      ctx.Channel,
		UserID:       ctx.UserID,
		UserName:     ctx.UserName,
		InstanceID:   ctx.InstanceID,
		InstanceName: ctx.InstanceName,
		InstanceIP:   ctx.InstanceIP,
		KeyUser:      ctx.KeyUser,
		RuleID:       hit.RuleID,
		RuleName:     hit.RuleName,
	}
	// 命令可能很长，按列宽截断而不是拒绝整条记录
	rec.Command = truncate(hit.Command, 1000)
	rec.Normalized = truncate(hit.Normalized, 1000)

	select {
	case blockLogCh <- rec:
	default:
		log.Printf("[高危指令拦截] 记录通道已满，本条留痕被丢弃（用户 %s，命令 %q）", ctx.UserName, rec.Command)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	// 命令里可能有中文，按字节截断后要退到完整字符边界
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}

// BlockLogQuery 拦截记录查询条件
type BlockLogQuery struct {
	PageNum      int
	PageSize     int
	UserName     string
	RuleName     string
	InstanceName string
	SessionID    string
}

// BlockLogList 分页查询高危指令拦截记录
func (s *DangerousCommandService) BlockLogList(q BlockLogQuery) (models.PageResult[models.OpsCommandBlockLog], error) {
	return models.Paginate[models.OpsCommandBlockLog](models.DB, q.PageNum, q.PageSize, func(db *gorm.DB) *gorm.DB {
		if q.UserName != "" {
			db = db.Where("user_name like ?", "%"+q.UserName+"%")
		}
		if q.RuleName != "" {
			db = db.Where("rule_name like ?", "%"+q.RuleName+"%")
		}
		if q.InstanceName != "" {
			db = db.Where("instance_name like ?", "%"+q.InstanceName+"%")
		}
		if q.SessionID != "" {
			db = db.Where("session_id = ?", q.SessionID)
		}
		return db.Order("id desc")
	})
}
