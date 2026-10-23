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
		{"rm  -rf /tmp/a", true},        // 多个空格
		{"rm -r -f /tmp/a", true},       // 拆开的短选项
		{"sudo rm -rf /tmp/a", true},    // 包装命令
		{"sudo  -u www-data rm -rf /", true},
		{"/bin/rm -rf /tmp/a", true},    // 绝对路径
		{"echo x; rm -rf /tmp/a", true}, // 分号后的第二段
		{"cat a | rm -rf /tmp", true},
		{`bash -c "rm -rf /tmp/a"`, true},
		{"FOO=bar rm -rf /tmp", true}, // 前导环境变量赋值
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

	// 方向键、Home 等只转发不进缓冲，编辑功能不受影响
	toRemote, hits := guard.Feed("ls\x1b[A\x1b[H\x1b[3~")
	if len(hits) != 0 {
		t.Fatalf("不应产生拦截：%+v", hits)
	}
	if !strings.Contains(string(toRemote), "\x1b[A") {
		t.Errorf("方向键序列应原样转发：%q", toRemote)
	}

	// 上一条安全命令被完整转发
	toRemote, hits = guard.Feed("whoami\r")
	if len(hits) != 0 {
		t.Fatalf("安全命令不该被拦：%+v", hits)
	}
	if string(toRemote) != "whoami\r" {
		t.Errorf("期望原样转发 whoami+回车，得到 %q", toRemote)
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
		in   string
		want int
	}{
		{"\x1b[A", 3},
		{"\x1b[200~x", 6},
		{"\x1b[1;5A", 6},
		{"\x1b[?2004h", 8},
		{"\x1bOP", 3},
		{"\x1b(B", 3},
		{"\x1b]0;title\x07rest", 10},
		{"\x1b", 1},
		{"\x1b[", 2},
	} {
		if got := escapeSeqLen(tc.in, 0); got != tc.want {
			t.Errorf("escapeSeqLen(%q)=%d，期望 %d", tc.in, got, tc.want)
		}
	}
}
