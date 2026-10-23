package instance

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"
	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// openScratchDB 用 .env 的 DSN 自建连接（不走 models.InitDB，因此不会触发 AutoMigrate），
// 数据库不可达时返回 nil 并由调用方跳过
func openScratchDB(t *testing.T) *gorm.DB {
	t.Helper()
	if strings.EqualFold(os.Getenv("DB_AUTOINIT"), "off") {
		t.Skip("DB_AUTOINIT=off，跳过需要数据库的测试")
		return nil
	}
	for _, p := range []string{".env", "../../.env"} {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
			break
		}
	}
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("未配置 DB_DSN，跳过")
		return nil
	}
	db, err := models.Open(dsn)
	if err != nil {
		t.Skipf("数据库不可达，跳过：%v", err)
		return nil
	}
	return db
}

// withRules 用给定的规则替换内存缓存，返回还原函数，使测试不依赖数据库
func withRules(t *testing.T, rules []*DangerousCommandRule) {
	t.Helper()
	rulesMutex.Lock()
	old := dangerousCommandRules
	dangerousCommandRules = rules
	rulesMutex.Unlock()
	t.Cleanup(func() {
		rulesMutex.Lock()
		dangerousCommandRules = old
		rulesMutex.Unlock()
	})
}

func regexRule(t *testing.T, name, pattern string) *DangerousCommandRule {
	t.Helper()
	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("正则编译失败 %s: %v", pattern, err)
	}
	return &DangerousCommandRule{
		Name: name, Pattern: pattern, MatchType: models.MatchTypeRegex, Regex: re,
	}
}

func rmRule() *DangerousCommandRule {
	return &DangerousCommandRule{ID: 15, Name: "删除命令", Pattern: "rm -rf", MatchType: models.MatchTypePrefix}
}

// TestCheckCommandPrefixRuleVariants 一条 `rm -rf` 前缀规则应当覆盖的常见改写形式
func TestCheckCommandPrefixRuleVariants(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})

	for _, tc := range []struct {
		cmd   string
		block bool
	}{
		{"rm -rf /tmp/a", true},
		{"  rm -rf /tmp/a  ", true},
		{"rm -rf", true},
		{"ls -l", false},
		// 归一化后必须覆盖的绕过形式
		{"rm  -rf /tmp/a", true},     // 多个空格
		{"rm -r -f /tmp/a", true},    // 拆开的短选项
		{"sudo rm -rf /tmp/a", true}, // 包装命令
		{"sudo  -u www-data rm -rf /", true},
		{"/bin/rm -rf /tmp/a", true},    // 绝对路径
		{"echo x; rm -rf /tmp/a", true}, // 分号后的第二段
		{"cat a | rm -rf /tmp", true},
		{`bash -c "rm -rf /tmp/a"`, true},
		{"FOO=bar rm -rf /tmp", true}, // 前导环境变量赋值
		// shell 保留字后面仍然是命令位置
		{"if [ -f /etc/passwd ]; then rm -rf /data; fi", true},
		{"for f in a b; do rm -rf /data; done", true},
		{"! rm -rf /data", true},
		{"while :; do rm -rf /data; done", true},
		// 引号自带的前导空格要压缩掉才能匹配上 `rm -rf` 前缀
		{`rm " -rf /data"`, true},
	} {
		got, _, _ := CheckCommand(tc.cmd, false)
		if got != tc.block {
			t.Errorf("CheckCommand(%q) 拦截=%v，期望=%v", tc.cmd, got, tc.block)
		}
	}
}

// TestCheckCommandNoFalsePositive 引号内的字面量参数不该被当成命令
func TestCheckCommandNoFalsePositive(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})

	for _, cmd := range []string{
		`grep "rm -rf" access.log`,
		`grep 'rm -rf' access.log`,
		"git commit -m 'fix rm -rf bug'",
		"echo done",
	} {
		if blocked, _, _ := CheckCommand(cmd, false); blocked {
			t.Errorf("引号内的字面量被误拦：%q", cmd)
		}
	}
}

func TestCheckCommandExactRuleStaysStrict(t *testing.T) {
	withRules(t, []*DangerousCommandRule{{ID: 9, Name: "停止系统", Pattern: "halt", MatchType: models.MatchTypeExact}})

	if blocked, _, _ := CheckCommand("halt", false); !blocked {
		t.Error("精确规则应命中原样输入")
	}
	// 精确匹配只对整行/整段生效，不下钻到任意 token 起点，避免误伤
	for _, cmd := range []string{"halt -t 5", "echo halt", "cat halt"} {
		if blocked, _, _ := CheckCommand(cmd, false); blocked {
			t.Errorf("精确规则不该命中 %q（旧语义即为整行相等）", cmd)
		}
	}
}

func TestCheckCommandRegexCoversVariants(t *testing.T) {
	withRules(t, []*DangerousCommandRule{regexRule(t, "删除根目录", `rm\s+-rf\s+/`)})

	for _, cmd := range []string{"rm -rf /", "rm  -rf  /var", "sudo rm -rf /", "echo hi; rm -rf /tmp"} {
		if blocked, _, _ := CheckCommand(cmd, false); !blocked {
			t.Errorf("正则规则应命中 %q（MatchString 不做锚定，前导命令与多空格都能覆盖）", cmd)
		}
	}
	// 相对路径不在该正则范围内，属于规则本身的选择而非匹配缺陷
	if blocked, _, _ := CheckCommand("rm -rf ./build", false); blocked {
		t.Error(`rm\s+-rf\s+/ 不应命中相对路径`)
	}
}

func TestCheckCommandAdminBypass(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	if blocked, _, _ := CheckCommand("rm -rf /", true); blocked {
		t.Error("管理员按设计应直接放行")
	}
}

func TestCheckCommandEmptyCacheAllowsAll(t *testing.T) {
	// 缓存为空时全部放行 —— 这正是启动未加载规则导致拦截「不生效」的表现
	withRules(t, nil)
	if blocked, _, _ := CheckCommand("rm -rf /", false); blocked {
		t.Error("空缓存不应拦截")
	}
}

func TestCheckCommandDetailedReturnsRuleIdentity(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	blocked, ruleID, ruleName, _, normalized := CheckCommandDetailed("sudo rm -rf /tmp", false)
	if !blocked || ruleID != 15 || ruleName != "删除命令" {
		t.Fatalf("期望返回规则身份，得到 blocked=%v id=%d name=%s", blocked, ruleID, ruleName)
	}
	if normalized != "rm -rf /tmp" {
		t.Errorf("期望返回命中形态 rm -rf /tmp，得到 %q", normalized)
	}
}

func TestNormalizeCommandKeepsRawFirst(t *testing.T) {
	got := NormalizeCommand("sudo rm -rf /tmp")
	if len(got) == 0 || got[0] != "sudo rm -rf /tmp" {
		t.Fatalf("第一个候选必须是原始输入，得到 %#v", got)
	}
}

// TestLineGuardEscapeSequenceNotBypassed 含 ESC 的输入块过去被整块透传、完全不校验
func TestLineGuardEscapeSequenceNotBypassed(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	guard := NewLineGuard(AuditContext{UserName: "ops", UserID: 74}, false)

	// 括号粘贴模式：一段带转义序列前缀、内部含回车的高危命令
	paste := "\x1b[200~rm -rf /\x1b[1;5A\r\x1b[201~"
	toRemote, hits := guard.Feed(paste)
	if len(hits) != 1 {
		t.Fatalf("粘贴的高危命令应被拦截，实际命中 %d 次，转发内容=%q", len(hits), toRemote)
	}
	if strings.Contains(string(toRemote), "\r") {
		t.Errorf("被拦截的行不应把回车转发给远端：%q", toRemote)
	}
	if !strings.Contains(string(toRemote), "\x03") {
		t.Errorf("应向远端发送 Ctrl+C 丢弃该行：%q", toRemote)
	}
}

func TestLineGuardForwardsControlKeysWithoutBuffering(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	guard := NewLineGuard(AuditContext{}, false)

	// 左右移动、Home/End、Delete 等编辑键照常转发且不进缓冲
	toRemote, hits := guard.Feed("ls\x1b[C\x1b[D\x1b[H\x1b[F\x1b[3~")
	if len(hits) != 0 {
		t.Fatalf("不应产生拦截：%+v", hits)
	}
	for _, want := range []string{"\x1b[C", "\x1b[D", "\x1b[H", "\x1b[F", "\x1b[3~"} {
		if !strings.Contains(string(toRemote), want) {
			t.Errorf("编辑键 %q 应原样转发：%q", want, toRemote)
		}
	}
	if strings.Contains(string(toRemote), "\x1b[3~l") {
		t.Errorf("Delete 不应把 l 删掉后再转发（本地要按同一语义镜像）：%q", toRemote)
	}

	// 上/下箭头不再转发：交给远端 readline 就会引入堡垒机看不见的召回内容
	if toRemote, _ = guard.Feed("ls\x1b[A"); strings.Contains(string(toRemote), "\x1b[A") {
		t.Errorf("上箭头不应原样转发，应由堡垒机接管：%q", toRemote)
	}

	// 安全命令原样转发（先 Ctrl+C 作废前面拼起来的测试残行）
	guard.Feed("\x03")
	toRemote, hits = guard.Feed("whoami\r")
	if len(hits) != 0 {
		t.Fatalf("安全命令不该被拦：%+v", hits)
	}
	if string(toRemote) != "whoami\r" {
		t.Errorf("期望原样转发 whoami+回车，得到 %q", toRemote)
	}
}

// TestLineGuardMetaAndControlKeysSwallowed Alt/Meta 组合键与未建模的控制字符都会让远端
// 在堡垒机看不见的情况下改动或执行当前行，一律不转发：
// M-p/M-n 历史搜索、M-d 按词删除、M-* glob 展开、Ctrl+O 直接执行当前行、
// Ctrl+V 转义插入、Ctrl+_ 撤销、Ctrl+X 前缀（Ctrl+X Ctrl+E 调出 $EDITOR 执行整条命令）
func TestLineGuardMetaAndControlKeysSwallowed(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})

	for _, in := range []string{
		"\x1bp", "\x1bn", "\x1bd", "\x1bb", "\x1bf", "\x1bu", "\x1b*", "\x1bt", "\x1b0", "\x1b(", "\x1b)",
		"\x0f", "\x14", "\x16", "\x1f", "\x1d", "\x1c",
	} {
		guard := NewLineGuard(AuditContext{}, false)
		out, _ := guard.Feed("echo x" + in)
		if strings.Contains(string(out), in) {
			t.Errorf("按键 %q 不应转发给远端：%q", in, out)
		}
	}

	// Ctrl+X 连同后一个字节一起吃掉，^X^E（调出 $EDITOR 执行整条命令）不会到达远端
	guard := NewLineGuard(AuditContext{}, false)
	out, _ := guard.Feed("echo x; \x18\x05")
	if strings.Contains(string(out), "\x18") || strings.HasSuffix(string(out), "\x05") {
		t.Errorf("Ctrl+X 及其绑定键应被吞掉：%q", out)
	}
	if _, hits := guard.Feed("rm -rf /data\r"); len(hits) != 1 {
		t.Errorf("Ctrl+X 之后打出的高危命令仍应被回车校验拦掉，实际 %v", hits)
	}

	// 管理员不受规则约束，也不接管按键（vim/top 这类全屏程序需要原始方向键）
	admin := NewLineGuard(AuditContext{}, true)
	out, hits := admin.Feed("rm -rf /\x1b[A\x1bd\r")
	if len(hits) != 0 {
		t.Errorf("管理员不应被拦：%+v", hits)
	}
	if string(out) != "rm -rf /\x1b[A\x1bd\r" {
		t.Errorf("管理员输入应原样透传，得到 %q", out)
	}
}

// TestLineGuardSplitEscapeSequence 一个按键被 TCP 包边界切开时，残缺序列不能进缓冲
func TestLineGuardSplitEscapeSequence(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	guard := NewLineGuard(AuditContext{}, false)

	guard.Feed("rm\r") // hist: rm

	// `ESC [` 先到、`A` 后到：远端收到的是完整上箭头，缓冲里也必须是召回结果而不是 "[A"
	if out, _ := guard.Feed("\x1b["); string(out) != "" {
		t.Errorf("残缺序列应等待下一段，不应提前转发：%q", out)
	}
	out, _ := guard.Feed("A")
	if string(out) != "rm" {
		t.Errorf("拼完整序列后应执行历史召回，得到 %q", out)
	}
	if _, hits := guard.Feed(" -rf /data\r"); len(hits) != 1 {
		t.Fatalf("召回后补参数必须被拦截，实际 %v", hits)
	}
}

// TestLineGuardLineContinuationCannotBypass 行尾反斜杠、未闭合引号、块关键字都会让远端
// 停在 PS2 续行提示符上，真正执行的是拼起来的整条命令，所以必须攒完再校验
func TestLineGuardLineContinuationCannotBypass(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})

	run := func(name, first, rest string, wantHits int) {
		t.Run(name, func(t *testing.T) {
			guard := NewLineGuard(AuditContext{}, false)
			out, hits := guard.Feed(first + "\r")
			if len(hits) != 0 {
				t.Fatalf("半条命令不该被拦：%+v（%q）", hits, out)
			}
			if !strings.Contains(string(out), "\r") {
				t.Errorf("续行的回车应照常转发：%q", out)
			}
			out, hits = guard.Feed(rest + "\r")
			if len(hits) != wantHits {
				t.Fatalf("续行拼完后应命中 %d 次，实际 %d 次，转发内容=%q", wantHits, len(hits), out)
			}
			if wantHits > 0 && strings.Contains(string(out), "\r") {
				t.Errorf("被拦的整条命令不应把回车发给远端：%q", out)
			}
		})
	}

	run("行尾反斜杠", `rm \`, "-rf /data", 1)
	run("未闭合双引号", `rm "`, `-rf /data"`, 1)
	run("未闭合括号", `echo $(rm`, " -rf /data)", 1)
	run("块关键字结尾", "if [ -f /etc/passwd ]; then", "rm -rf /data; fi", 1)
	run("悬空管道", "echo x |", " xargs rm -rf /data", 1)
	run("续行后仍安全", "echo \\", "hello", 0)
	run("落单反斜杠不是续行", `echo "a\\"`, "", 0)
}

// TestLineGuardCSIuEnterCannotBypass 终端把回车重映射成 CSI-u/带修饰的 Fn 形式时，
// 远端可能已经执行了这一行，必须按回车校验
func TestLineGuardCSIuEnterCannotBypass(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})

	for _, eol := range []string{"\x1b[13u", "\x1b[13;5u", "\x1b[27;5;13~", "\x1b[10;5u"} {
		guard := NewLineGuard(AuditContext{}, false)
		out, hits := guard.Feed("rm -rf /data" + eol)
		if len(hits) != 1 {
			t.Errorf("组合键 %q 等价于回车，必须拦截，实际 %v（%q）", eol, hits, out)
			continue
		}
		if strings.Contains(string(out), eol) {
			t.Errorf("被拦的回车序列不应转发：%q", out)
		}
	}

	// 普通功能键（F5）不是回车，不该触发校验
	guard := NewLineGuard(AuditContext{}, false)
	if _, hits := guard.Feed("ls\x1b[15~"); len(hits) != 0 {
		t.Errorf("F5 不该被当成回车：%+v", hits)
	}
}

// TestLineGuardOverlongLineFailsClosed 超出缓冲上限的字节进不了缓冲，
// 远端那一行已经不是堡垒机能校验的内容，回车一律拦掉
func TestLineGuardOverlongLineFailsClosed(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	guard := NewLineGuard(AuditContext{}, false)

	long := strings.Repeat("a", maxGuardBuffer) + " rm -rf /data"
	out, hits := guard.Feed(long + "\r")
	if len(hits) != 1 {
		t.Fatalf("超长行应被拦掉，实际 %v", hits)
	}
	if !strings.Contains(string(out), "\x03") {
		t.Errorf("应向远端发 Ctrl+C 丢弃整行：%q", out)
	}
	if hits[0].RuleName != "命令行超限" {
		t.Errorf("拦截原因应是命令行超限，得到 %q", hits[0].RuleName)
	}
	// 作废后恢复正常
	guard.Feed("\x03")
	if _, hits := guard.Feed("ls\r"); len(hits) != 0 {
		t.Errorf("超限状态应随 Ctrl+C 清除：%+v", hits)
	}
}

// TestLineGuardHistoryRecallCannotBypass 用户报告的实际绕过路径：
// 先输 `rm` 回车（本身不违规），再按上箭头把 `rm` 召回成当前行，补 ` -rf /path` 回车。
// 召回的文本是远端补上并回显的，堡垒机看不见，旧实现里回车时校验的只有 ` -rf /path`。
// 现在历史由 LineGuard 自己维护，召回结果会以字面字节重新打给远端，缓冲与那一行始终一致。
func TestLineGuardHistoryRecallCannotBypass(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	guard := NewLineGuard(AuditContext{UserName: "ops", UserID: 74}, false)

	// 第一步：`rm` 单独一条命令，放行并进会话历史
	toRemote, hits := guard.Feed("rm\r")
	if len(hits) != 0 {
		t.Fatalf("rm 本身不在规则内，不应拦截：%+v", hits)
	}
	if string(toRemote) != "rm\r" {
		t.Fatalf("期望原样转发，得到 %q", toRemote)
	}

	// 第二步：上箭头召回 `rm`，堡垒机把这条命令当作输入重新发出去
	toRemote, hits = guard.Feed("\x1b[A")
	if len(hits) != 0 {
		t.Fatalf("召回不该触发拦截：%+v", hits)
	}
	if string(toRemote) != "rm" {
		t.Errorf("上箭头应改写成重新发送 rm，得到 %q", toRemote)
	}

	// 第三步：补上参数再回车，此时必须拦住
	toRemote, hits = guard.Feed(" -rf /data\r")
	if len(hits) != 1 {
		t.Fatalf("召回后补参数必须被拦截，实际命中 %d 次，缓冲=%q", len(hits), toRemote)
	}
	if hits[0].Command != "rm -rf /data" {
		t.Errorf("命中记录应是完整命令，得到 %q", hits[0].Command)
	}
	if strings.Contains(string(toRemote), "\r") {
		t.Errorf("被拦的行不应把回车发给远端：%q", toRemote)
	}
}

// TestLineGuardRecallSemantics 覆盖方向键接管的各个分支：Ctrl-P/Ctrl-N 等价、
// 多次上箭头回溯、下箭头回到更新的一行、召回后再编辑、被拦的行不进历史
func TestLineGuardRecallSemantics(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	guard := NewLineGuard(AuditContext{}, false)

	feed := func(in string) (string, int) {
		out, hits := guard.Feed(in)
		return string(out), len(hits)
	}

	// 没有历史时按上箭头：什么都不发（宁可召回不出来，也不能引入看不见的内容）
	if got, _ := feed("\x1b[A"); got != "" {
		t.Errorf("空历史时上箭头应无输出，得到 %q", got)
	}

	feed("rm\r") // hist: rm
	feed("ls\r") // hist: rm, ls

	// 当前行是空的，召回不需要先清行
	if got, _ := feed("\x1b[A"); got != "ls" {
		t.Errorf("上箭头应发出 ls，得到 %q", got)
	}
	// 再按上箭头：先 Ctrl-A Ctrl-K 清掉当前行，再打上一条
	if got, _ := feed("\x1b[A"); got != "\x01\x0brm" {
		t.Errorf("连续上箭头应清行后回溯到 rm，得到 %q", got)
	}
	// 下箭头 → 回到更新的一条
	if got, _ := feed("\x1b[B"); got != "\x01\x0bls" {
		t.Errorf("下箭头应回到 ls，得到 %q", got)
	}
	// 再下箭头 → 空出新行
	if got, _ := feed("\x1b[B"); got != "\x01\x0b" {
		t.Errorf("下箭头到末尾应清空当前行，得到 %q", got)
	}
	// Ctrl+P / Ctrl+N 与方向键等价
	if got, _ := feed("\x10"); got != "ls" {
		t.Errorf("Ctrl+P 应等价于上箭头，得到 %q", got)
	}
	if got, _ := feed("\x0e"); got != "\x01\x0b" {
		t.Errorf("Ctrl+N 应等价于下箭头，得到 %q", got)
	}

	// 召回 rm 后接着补参数：回车时校验的是拼起来的整行，必须拦掉
	feed("\x03")
	if got, _ := feed("\x1b[A\x1b[A"); got != "ls\x01\x0brm" {
		t.Errorf("连续回溯应停在 rm，得到 %q", got)
	}
	if _, hits := feed(" -rf /data\r"); hits != 1 {
		t.Errorf("召回后补参数必须被拦截，实际 %v", hits)
	}

	// 被拦下的那条整行不进历史：再按上箭头拿到的还是 ls
	if got, _ := feed("\x1b[A"); got != "ls" {
		t.Errorf("被拦的命令不应进历史，召回应得到 ls，实际 %q", got)
	}
}

// TestLineGuardPositionalEditingKeepsSync 光标移到行首再输入，
// 本地镜像必须按同样的位置插入，否则校验的字符串和真正执行的字符串不同
func TestLineGuardPositionalEditingKeepsSync(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	guard := NewLineGuard(AuditContext{}, false)

	guard.Feed("x")
	guard.Feed("\x1b[D") // 左移一格，光标回到 x 之前
	guard.Feed("rm -rf /")
	if _, hits := guard.Feed("\r"); len(hits) != 1 {
		t.Fatalf("行首插入的高危命令必须被拦截（远端那一行是 rm -rf /x）")
	}

	// Ctrl+左/右这类没建模的按词移动：转发后立即重打整行来对齐
	guard.Feed("abc")
	out, _ := guard.Feed("\x1b[1;5D")
	if !strings.Contains(string(out), "\x01\x0babc") {
		t.Errorf("未建模的光标移动后应重打当前行，得到 %q", out)
	}
}

// TestLineGuardKillYankAndUTF8 覆盖 Ctrl-K/Ctrl-W/Ctrl-Y 与多字节退格
func TestLineGuardKillYankAndUTF8(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})

	guard := NewLineGuard(AuditContext{}, false)
	guard.Feed("rm -rf /tmp/x")
	guard.Feed("\x01") // Ctrl+A 到行首
	guard.Feed("\x0b") // Ctrl+K 杀掉整行（此时光标在行首）
	if _, hits := guard.Feed("\r"); len(hits) != 0 {
		t.Fatal("Ctrl+K 清空后回车应放行")
	}

	guard = NewLineGuard(AuditContext{}, false)
	guard.Feed("ls 删除")
	if _, hits := guard.Feed("\x7f\x7f\x7f\x7f\r"); len(hits) != 0 {
		t.Fatal("多字节退格不应触发拦截")
	}
	out, _ := guard.Feed("echo 中文")
	if strings.Contains(string(out), "\x03") {
		t.Errorf("普通输入不应被当成拦截：%q", out)
	}
	// 四个退格删掉「删除」两个 3 字节汉字；剩下的 "ls " 仍应留在缓冲里
	guard2 := NewLineGuard(AuditContext{}, false)
	guard2.Feed("rm -rf 中文")
	guard2.Feed("\x17") // Ctrl+W 杀掉前一个词（含多字节）
	if _, hits := guard2.Feed("\r"); len(hits) != 1 {
		t.Errorf("Ctrl+W 后剩 rm -rf 仍应命中前缀规则，实际 %v", hits)
	}

	// Ctrl+Y 把刚摘掉的词按字面字节重新打出去
	guard3 := NewLineGuard(AuditContext{}, false)
	guard3.Feed("ls abc")
	guard3.Feed("\x17")
	out, _ = guard3.Feed("\x19")
	if !strings.HasSuffix(string(out), "abc") {
		t.Errorf("Ctrl+Y 应重打摘掉的 abc，得到 %q", out)
	}
}

// TestLineGuardHistoryExpansionBlocked !! / !123 / !foo / ^a^b 执行的是展开后的命令，
// 堡垒机校验的只是占位符本身，必须整条拦掉；同时不能误伤逻辑取反和 !=
func TestLineGuardHistoryExpansionBlocked(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	guard := NewLineGuard(AuditContext{}, false)

	cases := map[string]bool{
		"!!":                                true,
		"!! /tmp":                           true,
		"!123":                              true,
		"!-2":                               true,
		"!rm":                               true,
		"echo !$":                           true,
		"^rm^ls":                            true,
		"!$ -rf /tmp":                       true,
		"ls":                                false,
		"if ! grep -q a b; then echo x; fi": false,
		"[ 1 != 2 ]":                        false,
		// bash 的历史展开发生在引号解析之前，`echo '!!'` 同样会展开，只能拦掉
		"echo '!!'": true,
	}
	for cmd, wantBlocked := range cases {
		g := NewLineGuard(AuditContext{}, false)
		_, hits := g.Feed(cmd + "\r")
		if got := len(hits) > 0; got != wantBlocked {
			t.Errorf("%q 期望拦截=%v，实际=%v", cmd, wantBlocked, got)
		}
	}

	// Ctrl+R 的反向搜索会把看不见的内容写进当前行，直接吞掉
	out, _ := guard.Feed("ls\x12")
	if strings.Contains(string(out), "\x12") {
		t.Errorf("Ctrl+R 不应转发给远端：%q", out)
	}
}

func TestLineGuardBackspaceAndCtrlC(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	guard := NewLineGuard(AuditContext{}, false)

	// 退格要同步收缩缓冲，否则删干净的回车仍会被判定为高危
	guard.Feed("rm -rf /")
	if _, hits := guard.Feed("\b\b\b\b\b\b\b\b"); len(hits) != 0 {
		t.Fatal("退格不应触发拦截")
	}
	if _, hits := guard.Feed("\r"); len(hits) != 0 {
		t.Fatal("退格清空后回车应放行")
	}

	// Ctrl+U 清行后回车同样放行
	guard.Feed("rm -rf /")
	guard.Feed("\x15")
	if _, hits := guard.Feed("\r"); len(hits) != 0 {
		t.Fatal("Ctrl+U 应清空当前行")
	}

	// Ctrl+C 作废当前行，且不该把被作废的行留在历史里
	guard.Feed("rm -rf /")
	guard.Feed("\x03")
	if _, hits := guard.Feed("\r"); len(hits) != 0 {
		t.Fatal("Ctrl+C 后回车应是空行")
	}
	if got, _ := guard.Feed("\x1b[A"); string(got) != "" {
		t.Errorf("Ctrl+C 作废的行不应进历史，得到 %q", got)
	}
}

// TestRecordCommandBlockPersists 验证拦截留痕真的落库（此前只写进程日志，事后无从追溯）。
// 需要 .env 中的数据库可达；用自建连接赋值 models.DB，避免触发 AutoMigrate。
func TestRecordCommandBlockPersists(t *testing.T) {
	db := openScratchDB(t)
	if db == nil {
		return
	}
	old := models.DB
	models.DB = db
	t.Cleanup(func() { models.DB = old })

	startBlockLogWriter()

	sessionID := fmt.Sprintf("ops-go-selftest-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		if err := db.Where("session_id = ?", sessionID).Delete(&models.OpsCommandBlockLog{}).Error; err != nil {
			t.Logf("清理测试留痕失败：%v", err)
		}
	})

	RecordCommandBlock(BlockHit{
		RuleID: 15, RuleName: "删除命令", Command: "rm -rf /tmp/selftest", Normalized: "rm -rf /tmp/selftest",
	}, AuditContext{
		SessionID: sessionID, Channel: models.BlockChannelBastion,
		UserID: 74, UserName: "selftest", InstanceID: 1, InstanceName: "自检主机", InstanceIP: "127.0.0.1",
		KeyUser: "root",
	})

	var count int64
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.Model(&models.OpsCommandBlockLog{}).Where("session_id = ?", sessionID).Count(&count).Error; err != nil {
			t.Fatalf("查询拦截记录失败：%v", err)
		}
		if count > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if count == 0 {
		t.Fatal("异步写入协程没有把拦截记录落库")
	}

	var rec models.OpsCommandBlockLog
	if err := db.Where("session_id = ?", sessionID).First(&rec).Error; err != nil {
		t.Fatalf("读取拦截记录失败：%v", err)
	}
	if rec.Command != "rm -rf /tmp/selftest" || rec.RuleName != "删除命令" || rec.UserName != "selftest" {
		t.Errorf("留痕字段写错：%+v", rec)
	}
}

func TestTruncateKeepsUTF8Boundary(t *testing.T) {
	s := strings.Repeat("a", 999) + "中文命令"
	got := truncate(s, 1000)
	if !utf8.ValidString(got) {
		t.Errorf("按字节截断产生了非法 UTF-8：%q", got[len(got)-4:])
	}
	if truncate("短", 1000) != "短" {
		t.Error("未超长的字符串不应被改动")
	}
}

func TestEscapeSeqLen(t *testing.T) {
	for _, tc := range []struct {
		in       string
		want     int
		complete bool
	}{
		{"\x1b[A", 3, true},
		{"\x1b[200~x", 6, true},
		{"\x1b[1;5A", 6, true},
		{"\x1b[?2004h", 8, true},
		{"\x1bOP", 3, true},
		{"\x1b(B", 3, true},
		{"\x1b]0;title\x07rest", 10, true},
		{"\x1bd", 2, true},
		{"\x1b[13;5u", 7, true},
		{"\x1b", 0, false},
		{"\x1b[", 0, false},
		{"\x1b]0;title", 0, false},
		{"\x1bO", 0, false},
	} {
		got, ok := escapeSeqLen(tc.in, 0)
		if got != tc.want || ok != tc.complete {
			t.Errorf("escapeSeqLen(%q)=%d/%v，期望 %d/%v", tc.in, got, ok, tc.want, tc.complete)
		}
	}
}
