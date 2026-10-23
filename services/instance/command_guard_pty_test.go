package instance

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// 这组测试把真实的交互式 bash 挂在 LineGuard 后面：单元测试只能证明字节改写逻辑自洽，
// 证不了 Ctrl-A/Ctrl-K 加字面重打在真 readline 上真的是「整行替换」、方向键接管之后
// 远端历史再也够不着。需要 util-linux 的 script 来分配 pty，没有就跳过。

type ptyGuard struct {
	t     *testing.T
	cmd   *exec.Cmd
	in    io.WriteCloser
	home  string
	guard *LineGuard

	mu  sync.Mutex
	out bytes.Buffer
}

const ptyPrompt = "OPS-GO-PTY> "

func startBashThroughGuard(t *testing.T, guard *LineGuard) *ptyGuard {
	t.Helper()
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("需要 util-linux 的 script 才能分配 pty")
	}

	work, err := os.MkdirTemp("", "ops-go-pty-")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	home, err := os.MkdirTemp("", "ops-go-pty-home-")
	if err != nil {
		t.Fatalf("创建临时 HOME 失败: %v", err)
	}
	// bash 启动后会把 PS1 覆盖成默认值，环境变量里的提示符不生效，只能用 rc 文件设
	rc := filepath.Join(home, "rc")
	rcBody := "PS1='" + ptyPrompt + "'\nPS2='PS2> '\n"
	if err := os.WriteFile(rc, []byte(rcBody), 0o600); err != nil {
		t.Fatalf("写入 rc 失败: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(work)
		os.RemoveAll(home)
	})

	cmd := exec.Command("script", "-qec", "bash --rcfile "+rc+" -i", "/dev/null")
	cmd.Dir = work
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"HISTFILE="+filepath.Join(home, "history"),
		"TERM=xterm",
		"PS1="+ptyPrompt,
		"PS2=PS2> ",
	)

	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("接管 stdin 失败: %v", err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("接管 stdout 失败: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动 bash 失败: %v", err)
	}

	p := &ptyGuard{t: t, cmd: cmd, in: in, home: home, guard: guard}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := out.Read(buf)
			if n > 0 {
				p.mu.Lock()
				p.out.Write(buf[:n])
				p.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	t.Cleanup(func() {
		_ = in.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	if !p.waitForPrompt(1) {
		t.Fatalf("bash 没有给出提示符，输出=%q", p.text())
	}
	return p
}

func (p *ptyGuard) text() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.out.String()
}

// waitForPrompt 等提示符出现 n 次（回车后 bash 真正回到了可输入状态）
func (p *ptyGuard) waitForPrompt(n int) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Count(p.text(), ptyPrompt) >= n {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// send 把按键交给 LineGuard，并把该转发的字节写进 pty；返回本次命中的拦截
func (p *ptyGuard) send(keys string) []BlockHit {
	p.t.Helper()
	before := strings.Count(p.text(), ptyPrompt)
	out, hits := p.guard.Feed(keys)
	if len(out) > 0 {
		if _, err := p.in.Write(out); err != nil {
			p.t.Fatalf("写入 pty 失败: %v", err)
		}
	}
	if len(hits) == 0 && strings.HasSuffix(keys, "\r") && p.guard.pending == "" {
		if !p.waitForPrompt(before + 1) {
			p.t.Fatalf("命令没有执行完，输出=%q", p.text())
		}
	} else {
		time.Sleep(120 * time.Millisecond)
	}
	return hits
}

// makeVictim 造一个待删目录，用于断言「高危命令到底有没有真的跑起来」
func makeVictim(t *testing.T) string {
	t.Helper()
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.MkdirAll(victim, 0o755); err != nil {
		t.Fatalf("创建 victim 失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(victim, "proof.txt"), []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("写入 proof.txt 失败: %v", err)
	}
	return victim
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TestPTYHistoryRecallCannotBypass 用户报告的绕过路径在真 readline 上的复现：
// 远端历史里预先塞一条 `rm -rf <victim>`，然后按上箭头召回并补回车。
// 方向键被接管后这个按键根本到不了远端，堡垒机自己召回出来的内容要能被校验。
func TestPTYHistoryRecallCannotBypass(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	victim := makeVictim(t)

	guard := NewLineGuard(AuditContext{}, false)
	p := startBashThroughGuard(t, guard)

	// 先把高危命令写进远端自己的历史列表：一旦方向键漏下去，一键就能召回它
	p.send("history -s 'rm -rf " + victim + "'\r")

	// 复现绕过：先执行一条本身无害的 rm，再上箭头召回、补参数、回车
	p.send("rm\r")
	if !exists(victim) {
		t.Fatalf("前置条件被破坏：victim 已被删除")
	}
	hits := p.send("\x1b[A")
	if len(hits) != 0 {
		t.Fatalf("召回不该触发拦截：%+v", hits)
	}
	hits = p.send(" -rf " + victim + "\r")
	if len(hits) != 1 {
		t.Errorf("召回后补参数必须被拦，实际 %v，输出=%q", hits, p.text())
	}
	if !exists(victim) {
		t.Error("被拦下的命令真的执行了：目录已不存在")
	}

	// 直接输入同样要被拦
	if hits = p.send("rm -rf " + victim + "\r"); len(hits) != 1 {
		t.Errorf("直接输入的高危命令应被拦截，实际 %v", hits)
	}
	if !exists(victim) {
		t.Error("直接输入的高危命令被执行了")
	}

	// 安全命令照常执行
	if hits = p.send("echo guard-ok\r"); len(hits) != 0 {
		t.Errorf("安全命令不该被拦：%+v", hits)
	}
	if !strings.Contains(p.text(), "guard-ok") {
		t.Errorf("安全命令没有执行，输出=%q", p.text())
	}
}

// TestPTYLineContinuationCannotBypass 行尾反斜杠续行：远端在 PS2 上还没执行，
// 真正跑起来的是拼接后的整条命令
func TestPTYLineContinuationCannotBypass(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	victim := makeVictim(t)

	guard := NewLineGuard(AuditContext{}, false)
	p := startBashThroughGuard(t, guard)

	hits := p.send("rm \\\r")
	if len(hits) != 0 {
		t.Fatalf("半条命令不该被拦：%+v", hits)
	}
	if !strings.Contains(p.text(), "PS2>") {
		t.Logf("期望远端停在 PS2 续行提示符，实际输出=%q", p.text())
	}
	if hits = p.send("-rf " + victim + "\r"); len(hits) != 1 {
		t.Errorf("续行拼完后应被拦截，实际 %v，输出=%q", hits, p.text())
	}
	if !exists(victim) {
		t.Error("续行拼接出的高危命令被执行了")
	}
}

// TestPTYEditingStillWorks 接管按键不能把普通编辑搞坏：光标左移后插入、退格、
// Ctrl-A/Ctrl-E、以及多字节字符都要和远端那一行一致
func TestPTYEditingStillWorks(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})

	guard := NewLineGuard(AuditContext{}, false)
	p := startBashThroughGuard(t, guard)

	// echo abcdef，光标左移一格再插一个 X → 期望输出 abcdeXf
	p.send("echo abcdef")
	p.send("\x1b[D")
	if hits := p.send("X\r"); len(hits) != 0 {
		t.Fatalf("编辑后的安全命令不该被拦：%+v", hits)
	}
	if !strings.Contains(p.text(), "abcdeXf") {
		t.Errorf("远端执行的内容与堡垒机校验的不一致，输出=%q", p.text())
	}

	// 打错后退格删掉，再输正常命令
	p.send("echo typoo")
	p.send("\x7f")
	if hits := p.send("s\r"); len(hits) != 0 {
		t.Fatalf("退格后的命令不该被拦：%+v", hits)
	}
	if !strings.Contains(p.text(), "typos") {
		t.Errorf("退格没有同步收缩缓冲，输出=%q", p.text())
	}

	// 中文参数原样转发
	if hits := p.send("echo 中文\r"); len(hits) != 0 {
		t.Fatalf("中文命令不该被拦：%+v", hits)
	}
	if !strings.Contains(p.text(), "中文") {
		t.Errorf("中文参数没有正确回显，输出=%q", p.text())
	}
}

// TestPTYMetaAndPrefixKeysSwallowed Alt 组合键与 Ctrl-X 绑定会在远端不可见地改行，
// 不能到达真 readline
func TestPTYMetaAndPrefixKeysSwallowed(t *testing.T) {
	withRules(t, []*DangerousCommandRule{rmRule()})
	victim := makeVictim(t)

	guard := NewLineGuard(AuditContext{}, false)
	p := startBashThroughGuard(t, guard)

	// 远端历史里同样先埋一条高危命令
	p.send("history -s 'rm -rf " + victim + "'\r")

	// M-p（history-search-backward）与 Ctrl+P 都不该把埋进去的命令带出来
	for _, keys := range []string{"\x1bp", "\x1b[1;5A", "\x18\x05"} {
		if hits := p.send("echo x" + keys + "\x03"); len(hits) != 0 {
			t.Errorf("按键 %q 不该触发拦截：%+v", keys, hits)
		}
		if !exists(victim) {
			t.Fatalf("按键 %q 让远端执行了埋进历史的高危命令", keys)
		}
	}
	// Ctrl-O 在 readline 里是「不经过回车直接执行当前行」
	if hits := p.send("rm -rf " + victim + "\x0f"); len(hits) != 0 {
		t.Logf("Ctrl+O 命中拦截：%+v", hits)
	}
	if !exists(victim) {
		t.Error("Ctrl+O 绕过回车校验执行了高危命令")
	}
}
