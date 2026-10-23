package instance

import (
	"fmt"
	"log"
	"regexp"
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

// maxPendingBytes 续行攒下的总量上限，超限后 fail-closed（回车时按拦截处理）
const maxPendingBytes = 8 * 1024

// 会话历史容量：召回是「堡垒机把内容重新打成字面字节发给远端」，所以条目本身也要限界，
// 否则用户可以靠反复输入超长命令把每个会话的内存撑大
const (
	maxHistoryEntries    = 100
	maxHistoryEntryBytes = 512
)

// LineGuard 把「逐字符累积 + 回车时校验高危规则」的终端输入逻辑收敛到一处，
// SSH 堡垒机与 Web 终端共用，避免两份实现行为漂移。
//
// 核心不变量：**远端那一行的内容永远等于 g.buf**。
// 只有满足这个不变量，回车时的校验才等价于真正要执行的命令，所以凡是「会让远端改动
// 当前行、而堡垒机看不到改了什么」的输入都必须被接管或吞掉，而不是原样转发：
//
//   - 旧实现遇到含 ESC 的输入块就整块透传、不进缓冲也不校验，粘贴一段命令
//     （括号粘贴模式带 \x1b[200~ 前缀）可直接绕过拦截。现在按序列解析：
//     转义序列只跳过它自己那几个字节，其余字节照旧走同一套缓冲与校验。
//   - 方向键上/下（以及 Ctrl+上/下、F1/F2、Ctrl-P/Ctrl-N）在远端 readline 里是
//     「历史召回」，召回的文本由远端补上并回显，字节走的是输出方向，堡垒机一个都收不到。
//     原样转发就会出现「屏幕上是一条高危命令、服务端缓冲里是另一条」的情况
//     （先输 `rm` 回车，再按上箭头补 ` -rf /path`，回车时校验的是 ` -rf`）。
//     因此历史召回改由 LineGuard 自己维护会话内历史，并把召回结果当作字面输入打给远端
//     （先 Ctrl-A、Ctrl-K 清掉当前行，再逐字发出去），远端 readline 的历史根本不参与。
//   - Ctrl-R（反向历史搜索）无法接管成等价行为，直接吞掉。
//   - 左右移动/删除类按键堡垒机按同样的语义在本地镜像一份（buf + 光标位置），
//     语义无法精确镜像的（如 Ctrl+左/右按词移动）转发后立刻重打一遍当前行来对齐。
//   - 一段输入被 TCP 包边界切开（`ESC [` 在一个包、`A` 在下一个包）时，尾部字节先存在
//     guard 里等下一段，否则残缺序列会被当作普通字符进缓冲，而远端收到的仍是完整按键。
//   - Alt/Meta 组合键（`ESC` + 可打印字符）在 readline 里是历史搜索(M-p/M-n)、按词删除
//     (M-d)、glob 展开(M-*)、大小写转换(M-u) 一类「远端改行、堡垒机看不见」的动作，一律不转发。
//   - 未建模的控制字符同样吞掉：Ctrl+O(operate-and-get-next) 会不经过回车直接执行当前行，
//     Ctrl+V(转义插入)、Ctrl+](字符搜索)、Ctrl+_/Ctrl+Z(撤销) 都会改动这一行。
//     Ctrl-X 是前缀键，其绑定（Ctrl+X Ctrl+E 调出 $EDITOR、Ctrl+X Ctrl+R 历史搜索、
//     Ctrl+X u 撤销）产生的内容全部不可见，连同后一个字节一起吞。
//   - 终端把回车重映射成 CSI-u 形式（Ctrl+Enter → `ESC[13;5u`、`ESC[27;5;13~`）时，
//     远端可能已经执行了这一行而堡垒机没收到回车，因此这类序列按回车处理。
//   - 行尾反斜杠、未闭合的引号/反引号/括号、悬空的 `|`/`&&`、以 then/do/in 结尾的块
//     都会让远端停在 PS2 续行提示符上：这些行不会立刻执行，最终执行的是把后续行拼起来的
//     整条命令，所以只校验单行等于放行。续行片段会攒起来（用空格拼接，归一化时每个命令
//     位置都展开成候选），直到整条命令写完才校验，命中就发 Ctrl+C 丢掉整段。
//   - 单行超过 maxGuardBuffer 时后面的字节进不了缓冲，此时 fail-closed：回车一律拦掉，
//     而不是拿一条被截断的、看不出问题的命令去校验。
//
// 管理员不受高危规则约束，行模型对他没有意义，而 vim/top/less 这类全屏程序又依赖方向键
// 与 Meta 键的原样行为，所以管理员输入直接透传。
//
// 仍然存在的缺口（不是这套行编辑镜像能解决的，见 todo-feature.md）：Tab 补全插入的文本、
// 嵌套程序（bash 子 shell、vi、mysql 客户端）里的输入、以及 `eval`/变量间接展开出的命令；
// 非管理员在远端全屏程序里的方向键会被历史召回接管。
type LineGuard struct {
	isAdmin bool
	ctx     AuditContext

	buf    []byte
	pos    int // 光标在 buf 中的字节偏移，与远端 readline 的光标保持一致
	hist   []string
	histAt int    // 正在编辑的历史项下标；等于 len(hist) 表示「新输入的一行」
	yanked []byte // Ctrl-W/U/K 摘掉的内容，供 Ctrl-Y 粘回

	pending  string // PS2 续行攒下的半条命令
	overflow bool   // 本条命令超出缓冲上限，尾部字节没进缓冲
	esccarry []byte // 被包边界切开的转义序列尾部，等下一段拼完
	skip     int    // 需要连后续字节一起吃掉的前缀键剩余长度（Ctrl-X）
}

func NewLineGuard(ctx AuditContext, isAdmin bool) *LineGuard {
	return &LineGuard{isAdmin: isAdmin, ctx: ctx}
}

// 转义序列在行镜像里对应的动作
type seqAction int

const (
	seqPass    seqAction = iota // 与当前行内容无关，原样转发
	seqSwallow                  // 会让远端不可见地改行，整条不转发
	seqSubmit                   // 终端把回车重映射成了别的序列，按回车处理
	seqPrev                     // 上一条历史命令
	seqNext                     // 下一条历史命令
	seqLeft                     // 光标左移
	seqRight                    // 光标右移
	seqHome                     // 光标到行首
	seqEnd                      // 光标到行尾
	seqDel                      // 删除光标处字符
	seqRealign                  // 会移动光标或改行但语义没建模：转发后重打整行对齐
)

// Feed 处理一段终端输入，返回需要写入远端 stdin 的字节（被拦截的行以 Ctrl+C 收尾）
// 以及本次命中的拦截事件，调用方负责把事件写日志/提示给用户端。
func (g *LineGuard) Feed(data string) ([]byte, []BlockHit) {
	if g.isAdmin {
		return []byte(data), nil
	}

	if len(g.esccarry) > 0 {
		data = string(g.esccarry) + data
		g.esccarry = g.esccarry[:0]
	}

	out := make([]byte, 0, len(data)+16)
	var hits []BlockHit

	for i := 0; i < len(data); i++ {
		ch := data[i]

		if g.skip > 0 {
			g.skip--
			continue
		}

		if ch == 0x1b {
			n, ok := escapeSeqLen(data, i)
			if !ok {
				// 序列还没收完：尾部留到下一段一起解析，剩下的字节已经全部属于这个序列
				g.esccarry = append(g.esccarry, data[i:]...)
				break
			}
			seq := data[i : i+n]
			i += n - 1

			switch interpretSeq(seq) {
			case seqPrev:
				out = append(out, g.recall(-1)...)
			case seqNext:
				out = append(out, g.recall(1)...)
			case seqLeft:
				g.moveCursor(-1)
				out = append(out, seq...)
			case seqRight:
				g.moveCursor(1)
				out = append(out, seq...)
			case seqHome:
				g.pos = 0
				out = append(out, seq...)
			case seqEnd:
				g.pos = len(g.buf)
				out = append(out, seq...)
			case seqDel:
				g.deleteForward()
				out = append(out, seq...)
			case seqRealign:
				// 语义没建模：转发后立刻按缓冲重打整行，把远端那一行拉回与 g.buf 相等
				out = append(out, seq...)
				out = append(out, g.retype()...)
			case seqSubmit:
				g.submitLine(&out, &hits, '\r')
			case seqSwallow:
			default:
				out = append(out, seq...)
			}
			continue
		}

		switch {
		case ch == '\r' || ch == '\n':
			g.submitLine(&out, &hits, ch)

		case ch == '\t':
			// Tab 补全结果由远端回显，内容不进缓冲（补全插入的文本堡垒机看不到，
			// 属于已知缺口，见类型注释）
			out = append(out, ch)

		case ch == 0x7f || ch == '\b':
			g.deleteBackward()
			out = append(out, ch)

		case ch == 0x03: // Ctrl+C：当前行作废，续行状态一起作废
			g.buf, g.pos = g.buf[:0], 0
			g.histAt = len(g.hist)
			g.pending, g.overflow = "", false
			out = append(out, ch)

		case ch == 0x15: // Ctrl+U：杀掉光标之前的内容
			g.killBackward()
			out = append(out, ch)

		case ch == 0x0b: // Ctrl+K：杀掉光标之后的内容
			g.killForward()
			out = append(out, ch)

		case ch == 0x17: // Ctrl+W：杀掉前一个词
			g.killWord()
			out = append(out, ch)

		case ch == 0x19: // Ctrl+Y：把摘掉的内容当作字面字节重新打出去（远端的 kill-ring 我们不参与）
			if len(g.yanked) > 0 {
				g.insert(g.yanked)
				out = append(out, g.yanked...)
			}

		case ch == 0x18: // Ctrl+X：readline 前缀键，它的绑定全部产生不可见内容，连后一个字节一起吃掉
			g.skip = 1

		case ch == 0x04: // Ctrl+D：空行时是 EOF（登出），有内容时是删除光标处字符
			if len(g.buf) > 0 {
				g.deleteForward()
			}
			out = append(out, ch)

		case ch == 0x01: // Ctrl+A：到行首
			g.pos = 0
			out = append(out, ch)

		case ch == 0x05: // Ctrl+E：到行尾
			g.pos = len(g.buf)
			out = append(out, ch)

		case ch == 0x02: // Ctrl+B：左移
			g.moveCursor(-1)
			out = append(out, ch)

		case ch == 0x06: // Ctrl+F：右移
			g.moveCursor(1)
			out = append(out, ch)

		case ch == 0x10: // Ctrl+P：上一条历史
			out = append(out, g.recall(-1)...)

		case ch == 0x0e: // Ctrl+N：下一条历史
			out = append(out, g.recall(1)...)

		case ch == 0x12: // Ctrl+R：反向历史搜索，召回内容不可见，直接吞掉
			continue

		default:
			if ch >= 0x20 {
				g.insert([]byte{ch})
				out = append(out, ch)
			} else if ch == 0x1a || ch == 0x11 || ch == 0x13 {
				// Ctrl+Z 挂起、Ctrl+Q/S 流量控制由远端 tty 驱动处理，不改动当前行内容
				out = append(out, ch)
			}
			// 其余控制字符（Ctrl+O 直接执行当前行、Ctrl+T 转置、Ctrl+V 转义插入、
			// Ctrl+] 字符搜索、Ctrl+_ 撤销……）没建模，转发就等于让远端偷偷改行，吞掉
		}
	}
	return out, hits
}

// submitLine 回车校验：单行命令直接校验；远端停在 PS2 续行时先把片段攒起来，
// 等整条命令写完再一起校验，命中就发 Ctrl+C 让远端把整段丢掉。
func (g *LineGuard) submitLine(out *[]byte, hits *[]BlockHit, eol byte) {
	line := strings.TrimSpace(string(g.buf))
	g.buf, g.pos = g.buf[:0], 0
	g.histAt = len(g.hist)

	cmd := line
	if g.pending != "" {
		// 用空格拼接：归一化会从每个命令位置展开候选，比按换行切分更严
		cmd = g.pending + " " + line
	}

	if cmd == "" {
		*out = append(*out, eol)
		return
	}

	if g.overflow {
		g.pending, g.overflow = "", false
		*out = append(*out, 0x03)
		*hits = append(*hits, BlockHit{
			RuleName:    "命令行超限",
			Description: fmt.Sprintf("单条命令超过 %d 字节，超出部分无法校验", maxGuardBuffer),
			Command:     cmd,
			Normalized:  cmd,
		})
		return
	}

	if unfinishedCommand(cmd) {
		if len(cmd) > maxPendingBytes {
			g.overflow = true
		}
		g.pending = stripLineContinuation(cmd)
		*out = append(*out, eol)
		return
	}
	g.pending = ""

	// 历史展开（!! / !123 / !foo / ^a^b）：回车后远端执行的是展开出来的命令，
	// 堡垒机校验的却是屏幕上这串占位符，等价于放行不可见内容，只能整条拦掉
	if historyExpansion(cmd) {
		*out = append(*out, 0x03)
		*hits = append(*hits, BlockHit{
			RuleName:    "历史展开",
			Description: "禁止通过 !! / !编号 / !前缀 或 ^a^b 引用历史命令，展开后的内容无法校验",
			Command:     cmd,
			Normalized:  cmd,
		})
		return
	}

	blocked, ruleID, ruleName, desc, normalized := CheckCommandDetailed(cmd, g.isAdmin)
	if !blocked {
		g.pushHistory(cmd)
		*out = append(*out, eol)
		return
	}
	// 只拦回车，让远端 shell 收到 Ctrl+C 丢弃这一整行
	*out = append(*out, 0x03)
	*hits = append(*hits, BlockHit{
		RuleID: ruleID, RuleName: ruleName, Description: desc,
		Command: cmd, Normalized: normalized,
	})
}

// 以下辅助方法都必须保持一个前提：它们改动 g.buf/g.pos 之后，发给远端的字节要让
// 远端那一行变成同样的内容。

func (g *LineGuard) insert(b []byte) {
	if len(g.buf)+len(b) > maxGuardBuffer {
		// 进不了缓冲的字节远端照样收得到，这一行已经不是堡垒机能校验的内容了：
		// 记下来，回车时按拦截处理（fail-closed），而不是拿截断后的样子去比对规则
		g.overflow = true
		return
	}
	// 先把光标之后的内容拷出来再拼接：直接 append(b, tail...) 会复用 b 的多余容量
	// （Ctrl+Y 粘的正是 guard 自己持有的 yanked 切片），把 yanked 写坏
	tail := append([]byte(nil), g.buf[g.pos:]...)
	g.buf = append(g.buf[:g.pos], b...)
	g.buf = append(g.buf, tail...)
	g.pos += len(b)
}

func (g *LineGuard) runeStartBefore(idx int) int {
	if idx <= 0 {
		return 0
	}
	j := idx - 1
	for j > 0 && !utf8.RuneStart(g.buf[j]) {
		j--
	}
	return j
}

// runeEndAfter 返回 idx 处那个 rune 的结束偏移（不含）
func (g *LineGuard) runeEndAfter(idx int) int {
	if idx >= len(g.buf) {
		return len(g.buf)
	}
	j := idx + 1
	for j < len(g.buf) && !utf8.RuneStart(g.buf[j]) {
		j++
	}
	return j
}

func (g *LineGuard) deleteBackward() {
	start := g.runeStartBefore(g.pos)
	g.yanked = append(g.yanked[:0], g.buf[start:g.pos]...)
	g.buf = append(g.buf[:start], g.buf[g.pos:]...)
	g.pos = start
}

func (g *LineGuard) deleteForward() {
	if g.pos >= len(g.buf) {
		return
	}
	end := g.runeEndAfter(g.pos)
	g.yanked = append(g.yanked[:0], g.buf[g.pos:end]...)
	g.buf = append(g.buf[:g.pos], g.buf[end:]...)
}

func (g *LineGuard) moveCursor(delta int) {
	g.pos += delta
	if g.pos < 0 {
		g.pos = 0
	}
	if g.pos > len(g.buf) {
		g.pos = len(g.buf)
	}
}

func (g *LineGuard) killBackward() {
	if g.pos == 0 {
		return
	}
	g.yanked = append(g.yanked[:0], g.buf[:g.pos]...)
	g.buf = append(g.buf[:0], g.buf[g.pos:]...)
	g.pos = 0
}

func (g *LineGuard) killForward() {
	if g.pos >= len(g.buf) {
		return
	}
	g.yanked = append(g.yanked[:0], g.buf[g.pos:]...)
	g.buf = g.buf[:g.pos]
}

func (g *LineGuard) killWord() {
	if g.pos == 0 {
		return
	}
	j := g.pos
	for j > 0 && g.buf[j-1] == ' ' {
		j--
	}
	for j > 0 && g.buf[j-1] != ' ' {
		j--
	}
	g.yanked = append(g.yanked[:0], g.buf[j:g.pos]...)
	g.buf = append(g.buf[:j], g.buf[g.pos:]...)
	g.pos = j
}

// recall 在会话历史里前后移动，并把召回结果写成远端当前行的内容。
// 原始按键不转发：交给远端 readline 就等于让它去查自己的历史，那份列表堡垒机看不见。
func (g *LineGuard) recall(delta int) []byte {
	if len(g.hist) == 0 {
		return nil
	}
	next := g.histAt + delta
	if next < 0 {
		next = 0
	}
	if next > len(g.hist) {
		next = len(g.hist)
	}
	g.histAt = next
	if next == len(g.hist) {
		return g.writeLine("")
	}
	return g.writeLine(g.hist[next])
}

// writeLine 把远端当前行替换成 text：先 Ctrl-A 回到行首、Ctrl-K 杀掉整行，
// 再把 text 当普通输入逐字节发出去，于是远端那一行与本地缓冲再次相等
func (g *LineGuard) writeLine(text string) []byte {
	out := make([]byte, 0, len(text)+2)
	if len(g.buf) > 0 {
		out = append(out, 0x01, 0x0b)
	}
	g.buf, g.pos = g.buf[:0], 0
	if len(text) > 0 {
		g.buf = append(g.buf, text...)
		g.pos = len(g.buf)
		out = append(out, text...)
	}
	return out
}

// retype 用当前缓冲重打一遍，用来在光标语义没建模时强制重新对齐
func (g *LineGuard) retype() []byte {
	return g.writeLine(string(g.buf))
}

func (g *LineGuard) pushHistory(cmd string) {
	entry := truncate(cmd, maxHistoryEntryBytes)
	if len(g.hist) >= maxHistoryEntries {
		g.hist = append(g.hist[:0], g.hist[1:]...)
	}
	g.hist = append(g.hist, entry)
	g.histAt = len(g.hist)
}

// historyExpansion 判断命令行里是否引用了历史（展开结果看不见）
func historyExpansion(cmd string) bool {
	for _, cand := range NormalizeCommand(cmd) {
		if reHistoryRef.MatchString(cand) {
			return true
		}
	}
	return false
}

// reHistoryRef 匹配 !!、!$、!*、!123、!-3、!命令前缀 以及 ^旧^新；
// 不会误伤 bash 的逻辑取反 `! cmd`、比较 `a != b`，也不会误伤 `#!/bin/sh`
var reHistoryRef = regexp.MustCompile(`!!|!(?:\$|\*|-?[0-9]+|[A-Za-z_][A-Za-z0-9_]*)|\^[^ \t]+\^[^ \t]*`)

// interpretSeq 把转义序列翻译成对当前行的动作。认不出来的一律按 seqRealign 处理：
// 转发后按缓冲重打整行——远端那一行最终仍等于 g.buf，序列本身做什么都无所谓。
func interpretSeq(seq string) seqAction {
	if len(seq) < 2 || seq[0] != 0x1b {
		return seqPass
	}
	rest := seq[1:]
	switch rest[0] {
	case '[':
		return interpretCSI(rest[1:])
	case 'O':
		// SS3：应用光标键模式下的方向键，以及 F1-F4
		if len(rest) == 2 {
			switch rest[1] {
			case 'A':
				return seqPrev
			case 'B':
				return seqNext
			case 'C':
				return seqRight
			case 'D':
				return seqLeft
			case 'P': // F1，xterm 默认绑定是历史搜索
				return seqPrev
			case 'Q': // F2
				return seqNext
			}
		}
		return seqRealign
	}
	// ESC + 可打印字符 = readline 的 Meta 键（M-p/M-n 历史搜索、M-d 按词删除、M-* glob 展开、
	// M-u 大写、M-b/M-f 按词移动…），效果全是远端偷偷改行；这些键在堡垒机里没有安全等价物，不转发
	if len(rest) == 1 && rest[0] >= 0x21 && rest[0] <= 0x7e {
		return seqSwallow
	}
	return seqPass
}

func interpretCSI(body string) seqAction {
	if body == "" {
		return seqRealign
	}
	final := body[len(body)-1]
	params := body[:len(body)-1]
	if final < 0x40 || final > 0x7e {
		return seqRealign
	}
	switch final {
	case 'A':
		return seqPrev
	case 'B':
		return seqNext
	case 'C':
		return motionAction(params, seqRight)
	case 'D':
		return motionAction(params, seqLeft)
	case 'H':
		return seqHome
	case 'F':
		return seqEnd
	case 'u':
		// xterm modifyOtherKeys 的 CSI-u 编码：ESC[<keysym>;<修饰>u。
		// 终端把 Ctrl+Enter 之类映射到回车 keysym 时，远端可能已经执行了这一行，
		// 而堡垒机一个回车都没收到 → 必须按回车走校验
		if keysym := csiFirst(params); keysym == 10 || keysym == 13 {
			return seqSubmit
		}
		return seqRealign
	case '~':
		switch params {
		case "3":
			return seqDel
		case "1":
			return seqHome
		case "4":
			return seqEnd
		case "200", "201":
			// 括号粘贴模式的首尾标记，与当前行内容无关
			return seqPass
		}
		// ESC[27;<修饰>;<keysym>~ 是另一种 Ctrl/Alt+组合键编码，keysym 是回车同样危险
		if nums := csiNumbers(params); len(nums) == 3 && (nums[2] == 10 || nums[2] == 13) {
			return seqSubmit
		}
		return seqRealign
	}
	return seqRealign
}

// csiFirst 取 CSI 参数的第一个数值
func csiFirst(params string) int {
	nums := csiNumbers(params)
	if len(nums) == 0 {
		return 0
	}
	return nums[0]
}

// csiNumbers 按 ';' 拆 CSI 参数并取数字，非纯数字字段（私有前缀 '?'、越界值）按 0 处理
func csiNumbers(params string) []int {
	out := make([]int, 0, 3)
	for _, p := range strings.Split(params, ";") {
		n := 0
		for i := 0; i < len(p); i++ {
			if p[i] < '0' || p[i] > '9' || n > 1<<16 {
				n = 0
				break
			}
			n = n*10 + int(p[i]-'0')
		}
		out = append(out, n)
	}
	return out
}

func motionAction(params string, plain seqAction) seqAction {
	if params == "" || params == "1" {
		return plain
	}
	// 带修饰键（Ctrl+左/右按词移动、Shift+左/右扩选）或多次移动没有建模，
	// 转发后重打整行保证与缓冲一致
	return seqRealign
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

// escapeSeqLen 返回 s[i]（必为 ESC）起始的终端序列长度。
// 第二个返回值 false 表示序列在这段输入结束处还没收完，调用方要把尾部留到下一段——
// 否则 `ESC [` 和 `A` 分在两个 TCP 包里时，残缺序列会被当成普通字符进缓冲，
// 而远端收到的是完整的上箭头，屏幕与缓冲从此错位。
func escapeSeqLen(s string, i int) (int, bool) {
	if i+1 >= len(s) {
		return 0, false
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
			return j + 1 - i, true
		}
		return 0, false
	case ']': // OSC: 以 BEL 或 ST(ESC \) 结束
		j := i + 2
		for j < len(s) {
			if s[j] == 0x07 {
				return j + 1 - i, true
			}
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2 - i, true
			}
			j++
		}
		return 0, false
	case 'P', 'X', '^', '_': // DCS/SOS/PM/APC：以 ST 结束
		j := i + 2
		for j < len(s) {
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2 - i, true
			}
			j++
		}
		return 0, false
	case 'O': // SS3: ESC O x
		if i+2 < len(s) {
			return 3, true
		}
		return 0, false
	case '(', ')', '#', '%': // 字符集指定序列，ESC + 选择符 + 终字符共 3 字节
		if i+2 < len(s) {
			return 3, true
		}
		return 0, false
	default:
		if s[i+1] >= 0x20 && s[i+1] <= 0x7e {
			// Alt/Meta 组合键
			return 2, true
		}
		// 单独的 ESC（后面跟控制字符或什么都没有）
		return 1, true
	}
}

// unfinishedCommand 判断「这一行回车后远端是否还没执行」，也就是 readline/bash 会不会
// 给出 PS2 续行提示符。命中任意一条即认为命令还没写完：
//   - 引号、反引号、()、[]、{} 没闭合
//   - 行尾落单的反斜杠（bash 把下一行直接拼进来，反斜杠和换行一起消失）
//   - 悬空的重定向或逻辑运算符（`>`、`|`、`&&`、`||`）
//   - 以 then/do/in 这类块关键字结尾
//
// 这些形态必须攒完整段再校验：单看 `rm \` 或 `-rf /` 都不像危险命令，
// 拼起来才是要执行的那一条。
func unfinishedCommand(cmd string) bool {
	var stack []byte
	quote := byte(0)
	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		if quote == '\'' {
			// 单引号里一切皆字面量
			if ch == '\'' {
				quote = 0
			}
			continue
		}
		if ch == '\\' {
			// 反斜杠转义下一个字节；行尾落单的反斜杠是续行
			if i+1 >= len(cmd) {
				return true
			}
			i++
			continue
		}
		if quote == '"' {
			switch {
			case ch == '"':
				quote = 0
			case ch == '`':
				stack = pushBrace(stack, ch)
			case ch == '$' && i+1 < len(cmd) && cmd[i+1] == '(':
				stack = append(stack, '(')
				i++
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
		case '`', '(', '[', '{':
			stack = pushBrace(stack, ch)
		case ')', ']', '}':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if quote != 0 || len(stack) > 0 {
		return true
	}

	tail := strings.TrimRight(cmd, " \t")
	if tail == "" {
		return false
	}
	// 单独的 & 是后台执行，属于已经写完
	switch {
	case strings.HasSuffix(tail, "|"),
		strings.HasSuffix(tail, "&&"),
		strings.HasSuffix(tail, "||"),
		strings.HasSuffix(tail, ">"),
		strings.HasSuffix(tail, "<"):
		return true
	}
	last := tail
	if j := strings.LastIndexAny(last, " \t;|&("); j >= 0 {
		last = last[j+1:]
	}
	switch last {
	case "then", "do", "in", "else", "elif", "case", "select":
		return true
	}
	return false
}

// pushBrace 反引号成对出现（命令替换），括号按类型配对即可，这里只关心「有没有闭合」
func pushBrace(stack []byte, ch byte) []byte {
	if ch == '`' && len(stack) > 0 && stack[len(stack)-1] == '`' {
		return stack[:len(stack)-1]
	}
	return append(stack, ch)
}

// stripLineContinuation 去掉行尾落单的反斜杠：续行拼进同一条命令时反斜杠本身消失
func stripLineContinuation(cmd string) string {
	tail := strings.TrimRight(cmd, " \t")
	n := 0
	for i := len(tail) - 1; i >= 0 && tail[i] == '\\'; i-- {
		n++
	}
	if n%2 == 1 {
		return tail[:len(tail)-1]
	}
	return cmd
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
