package instance

import (
	"strings"
)

// 本文件解决「规则配了但命令变体绕过」的问题。
// 旧实现只对 TrimSpace 后的整行做 精确/前缀/正则 匹配，于是
// `sudo  rm   -r -f /tmp`、`echo x; rm -rf /tmp`、`bash -c "rm -rf /tmp"` 都能穿过
// 一条 pattern 为 `rm -rf` 的前缀规则。归一化只做「额外候选形态」的展开，
// 规则本身是否命中仍由规则的 match_type 决定。

// shellReservedWords shell 保留字：它们出现在命令位置时，后面跟着的还是命令，
// 不能像普通命令那样「走到真命令就停止下钻」。少了这一步，
// `if [ -f x ]; then rm -rf /`、`! rm -rf /`、`for f in a; do rm -rf /; done` 都拦不住
var shellReservedWords = map[string]bool{
	"then": true, "else": true, "elif": true, "fi": true, "do": true, "done": true,
	"in": true, "esac": true, "case": true, "select": true, "until": true,
	"while": true, "for": true, "if": true, "!": true, "function": true,
}

// shellSeparators 顶层命令分隔符（引号内不算）
const shellSeparators = ";|&\n\r(){}"

// commandWrappers 这些前导词只是包装执行，需要剥掉后才能暴露真正的命令
var commandWrappers = map[string]bool{
	"sudo": true, "doas": true, "su": true, "runuser": true, "pkexec": true,
	"env": true, "nohup": true, "exec": true, "time": true, "timeout": true,
	"nice": true, "ionice": true, "setsid": true, "watch": true,
	"command": true, "builtin": true, "busybox": true, "xargs": true,
	"sh": true, "bash": true, "dash": true, "zsh": true, "ksh": true, "csh": true, "fish": true,
}

// shellWithCommandFlag 用 -c 携带一段脚本的解释器/xargs，需要递归展开它的内容
var shellWithCommandFlag = map[string]bool{
	"sh": true, "bash": true, "dash": true, "zsh": true, "ksh": true, "csh": true, "fish": true,
	"xargs": true, "watch": true, "timeout": true, "nohup": true, "env": true,
}

// wrapperValueFlags 包装命令会吃掉后一个 token 的选项（-u 用户、-c 脚本 等）
var wrapperValueFlags = map[string]bool{
	"-u": true, "-g": true, "-U": true, "-c": true, "-l": true, "-s": true,
	"-t": true, "-p": true, "-n": true, "-d": true, "-I": true, "-i": true, "-e": true,
}

// maxNormalizeDepth 递归展开 `bash -c "sh -c '...'"` 的最大层数
const maxNormalizeDepth = 3

// maxCandidates 单条命令行最多产生的候选形态数，防止超长输入放大匹配开销
const maxCandidates = 512

// cmdToken 一个词，保留它原来是否被引号包裹
type cmdToken struct {
	text   string
	quoted bool
}

// tokenizeTop 引号感知的切分：引号内的空格与分隔符都不参与切分，
// 返回值按顶层分隔符分组（组内是该段命令的 token 序列）。
func tokenizeTop(s string) [][]cmdToken {
	var groups [][]cmdToken
	var cur []cmdToken
	var sb strings.Builder
	quoted := false
	quoteCh := byte(0)
	flushToken := func() {
		if sb.Len() > 0 {
			cur = append(cur, cmdToken{text: sb.String(), quoted: quoted})
			sb.Reset()
		}
		quoted = false
		quoteCh = 0
	}
	flushGroup := func() {
		flushToken()
		if len(cur) > 0 {
			groups = append(groups, cur)
			cur = nil
		}
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case quoted:
			if ch == quoteCh {
				// 引号结束，但 token 继续（如 a"b"c）
				quoted = false
				quoteCh = 0
				continue
			}
			sb.WriteByte(ch)
		case ch == '\'' || ch == '"':
			// 引号开始：先落盘前缀，后续内容标记为字面量
			flushToken()
			quoted = true
			quoteCh = ch
		case ch == '\\' && i+1 < len(s):
			i++
			sb.WriteByte(s[i])
		case ch == '$' && i+1 < len(s) && s[i+1] == '(':
			// $( ... ) 命令替换：当作分隔符，内部命令单独成组
			flushGroup()
			i++
		case strings.IndexByte(shellSeparators, ch) >= 0 || ch == ' ' || ch == '\t':
			flushToken()
			if strings.IndexByte(shellSeparators, ch) >= 0 {
				flushGroup()
			}
		default:
			sb.WriteByte(ch)
		}
	}
	flushGroup()
	return groups
}

// basename 去掉命令的目录前缀：/usr/bin//rm -> rm
func basename(cmd string) string {
	if i := strings.LastIndexByte(cmd, '/'); i >= 0 {
		return cmd[i+1:]
	}
	return cmd
}

// isEnvAssign 判断是否为 FOO=bar 形式的环境变量赋值
func isEnvAssign(t cmdToken) bool {
	if t.quoted {
		return false
	}
	i := strings.IndexByte(t.text, '=')
	return i > 0 && strings.IndexFunc(t.text[:i], func(r rune) bool {
		return !(r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'))
	}) < 0
}

// isShortFlag -x 或 -xyz 形式
func isShortFlag(t cmdToken) bool {
	if t.quoted || len(t.text) < 2 || t.text[0] != '-' || t.text[1] == '-' {
		return false
	}
	return strings.TrimLeft(t.text, "-") == t.text[1:]
}

// mergeFlagsVariant 把拆开的短选项并回去：rm -r -f /tmp -> rm -rf /tmp
func mergeFlagsVariant(toks []cmdToken) []cmdToken {
	out := make([]cmdToken, 0, len(toks))
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		out = append(out, t)
		if !isShortFlag(t) || i+1 >= len(toks) || !isShortFlag(toks[i+1]) {
			continue
		}
		// 连续的短选项合并为一个 token，同时回退掉刚写入的那个
		j := i
		for j+1 < len(toks) && isShortFlag(toks[j+1]) {
			j++
		}
		merged := strings.TrimLeft(t.text, "-")
		for k := i + 1; k <= j; k++ {
			merged += strings.TrimLeft(toks[k].text, "-")
		}
		out[len(out)-1] = cmdToken{text: "-" + merged, quoted: true}
		i = j
	}
	return out
}

func joinToks(toks []cmdToken) string {
	parts := make([]string, 0, len(toks))
	for _, t := range toks {
		parts = append(parts, t.text)
	}
	return strings.Join(parts, " ")
}

// collapseSpaces 把连续空白压成一个空格。引号内的内容参与 token 拼接后容易出现
// `rm  -rf /data`（引号自带前导空格），不压缩的话 `rm -rf` 这类前缀规则匹配不上。
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// NormalizeCommand 返回一条命令参与高危匹配的全部候选形态，第一个始终是原始输入，
// 以保证旧的精确匹配语义不被削弱。
func NormalizeCommand(raw string) []string {
	raw = strings.TrimSpace(raw)
	seen := map[string]bool{}
	out := make([]string, 0, 16)
	add := func(s string) bool {
		s = collapseSpaces(s)
		if s == "" || seen[s] || len(out) >= maxCandidates {
			return false
		}
		seen[s] = true
		out = append(out, s)
		return true
	}
	add(raw)

	var walk func(toks []cmdToken, depth int)
	walk = func(toks []cmdToken, depth int) {
		if len(toks) == 0 {
			return
		}
		// 合并拆开的短选项后也作为一条命令处理
		if merged := mergeFlagsVariant(toks); len(merged) != len(toks) {
			add(joinToks(merged))
			walk(merged, depth)
		}

		// 从每个「命令位置」开始的后缀都是一种候选形态：
		// sudo -u www-data rm -rf /tmp 会依次暴露到 rm -rf /tmp
		for i := 0; i < len(toks); i++ {
			t := toks[i]
			add(joinToks(toks[i:]))

			// 绝对路径命令补一份去目录的形态：/bin/rm -rf /tmp -> rm -rf /tmp
			if !t.quoted {
				if n := basename(t.text); n != t.text && n != "" {
					alt := make([]cmdToken, len(toks)-i)
					copy(alt, toks[i:])
					alt[0] = cmdToken{text: n}
					add(joinToks(alt))
				}
			}

			if t.quoted {
				// 引号内是字面量参数（如 grep "rm -rf"），不作为命令起点继续下钻
				return
			}
			name := basename(t.text)
			if isEnvAssign(t) {
				continue
			}
			if strings.HasPrefix(t.text, "-") {
				// 包装命令的选项；带值选项把下一个 token 一起吃掉
				if wrapperValueFlags[name] {
					i++
				}
				continue
			}
			if shellReservedWords[name] {
				// 保留字之后还是命令位置，继续往后展开
				continue
			}
			if !commandWrappers[name] {
				// 走到这里说明 t 就是真正的命令，后缀展开到此为止
				return
			}
			// 包装命令：跳过它，继续看后面的部分
			if shellWithCommandFlag[name] && i+1 < len(toks) {
				next := toks[i+1]
				if !next.quoted && (next.text == "-c" || next.text == "-C") && i+2 < len(toks) {
					// sh -c "内联脚本"：把脚本内容当独立命令行再展开一层
					if depth < maxNormalizeDepth {
						for _, g := range tokenizeTop(toks[i+2].text) {
							walk(g, depth+1)
						}
					}
				}
			}
		}
	}

	for _, group := range tokenizeTop(raw) {
		walk(group, 0)
	}
	return out
}
