package bastion

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"

	sshclient "golang.org/x/crypto/ssh"
)

// TestKeyboardAuthPromptsNotEchoed 钉住「堡垒机口令提示不得要求客户端回显」。
// keyboard-interactive 的 echo 标志由服务端下发，一旦传 true，OpenSSH/PuTTY 就会把用户敲的
// 口令明文显示在屏幕上（历史上第一问就误传过 true），而这一点只有连上真实监听端口才观测得到。
// 探针在回调里读完标志就返回错误中止，不提交任何口令，因此不触发密码比对、失败计数与封禁。
// 堡垒机未运行或探测用户不存在（拿不到 prompt）时跳过。
// 本测试故意不加载 .env：go test 的工作目录是包目录（bastion/），models 的 init 连不上库只打日志，
// 因此不会触发 AutoMigrate，也不会碰生产数据。
func TestKeyboardAuthPromptsNotEchoed(t *testing.T) {
	port := os.Getenv("SSH_PORT")
	if port == "" {
		port = "2222"
	}
	user := os.Getenv("BASTION_PROBE_USER")
	if user == "" {
		user = "ops"
	}
	addr := net.JoinHostPort("127.0.0.1", port)

	probe, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Skipf("堡垒机端口 %s 不可达，跳过（%v）", addr, err)
	}
	probe.Close()

	type prompt struct {
		question    string
		instruction string
		echo        bool
	}
	var got []prompt

	cfg := &sshclient.ClientConfig{
		User: user,
		Auth: []sshclient.AuthMethod{
			sshclient.KeyboardInteractive(func(_, instruction string, questions []string, echos []bool) ([]string, error) {
				for i, q := range questions {
					got = append(got, prompt{question: q, instruction: instruction, echo: i < len(echos) && echos[i]})
				}
				return nil, errors.New("测试探针：只读取回显标志，不提交口令")
			}),
		},
		HostKeyCallback: sshclient.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	if conn, err := sshclient.Dial("tcp", addr, cfg); err == nil {
		conn.Close()
	}

	if len(got) == 0 {
		t.Skipf("用户 %s 未收到任何 prompt（用户不存在或被禁用），跳过", user)
	}
	for _, p := range got {
		if p.echo {
			t.Errorf("prompt %q（instruction %q）要求客户端回显输入，口令会显示在屏幕上", p.question, p.instruction)
		}
	}
	t.Logf("共取得 %d 个 prompt，均不要求回显", len(got))
}
