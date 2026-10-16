package datasource

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// 端到端用例需要本机 Loki/ssh，默认跳过，避免无依赖环境误跑：
// LOKI_E2E=1 go test -run TestLoki -v ./services/logs/datasource

func TestTailParseOutput(t *testing.T) {
	out := "OPS_META inode=123 pos=45 size=45 dropped=2 file=/var/log/a.log\nboot\nERROR boom\n"
	result, err := ParseTailOutput(out)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.Inode != 123 || result.Pos != 45 || result.Dropped != 2 {
		t.Fatalf("元信息解析错误: %+v", result)
	}
	if result.FilePath != "/var/log/a.log" {
		t.Fatalf("文件名解析错误: %s", result.FilePath)
	}
	if len(result.Lines) != 2 || result.Lines[1] != "ERROR boom" {
		t.Fatalf("行内容解析错误: %#v", result.Lines)
	}

	// 末行无换行：剔除半行并回退位点，等下一轮补齐
	fragmented, err := ParseTailOutput("OPS_META inode=1 pos=15 size=15 dropped=0 file=/a\nfull\nhalf-trunc")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(fragmented.Lines) != 1 || fragmented.Lines[0] != "full" {
		t.Fatalf("半行处理错误: %#v", fragmented.Lines)
	}
	if fragmented.Pos != 5 || !fragmented.HalfLine {
		t.Fatalf("半行位点回退错误: pos=%d half=%v", fragmented.Pos, fragmented.HalfLine)
	}
}

func TestTailErrorMessage(t *testing.T) {
	if _, err := ParseTailOutput("OPS_ERR 日志文件不存在: /a"); err == nil {
		t.Fatal("应识别远程错误")
	}
	// 错误原因会直接展示在采集源列表，需去掉前缀且不留多余空格
	_, err := ParseTailOutput("OPS_ERR  日志文件不可读(需要 root 或加入 adm 组): /var/log/syslog\n")
	if err == nil {
		t.Fatal("应识别远程错误")
	}
	want := "远程采集失败: 日志文件不可读(需要 root 或加入 adm 组): /var/log/syslog"
	if err.Error() != want {
		t.Errorf("错误原因文案不规范:\n got=%s\nwant=%s", err.Error(), want)
	}
}

func TestRunTailScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	if err := os.WriteFile(path, []byte("line1 INFO ok\nline2 ERROR boom\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result := runTail(t, path, 0, 0, 1<<20, 0)
	if len(result.Lines) != 2 {
		t.Fatalf("首次读取行数错误: %#v", result.Lines)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("line3 WARN slow\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	incremental := runTail(t, path, result.Pos, result.Inode, 1<<20, 0)
	if len(incremental.Lines) != 1 || incremental.Lines[0] != "line3 WARN slow" {
		t.Fatalf("增量读取内容错误: %#v", incremental.Lines)
	}

	// 轮转：新 inode 应重新从头读取
	rotated := filepath.Join(dir, "app.log.1")
	if err := os.Rename(path, rotated); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fresh after rotation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	afterRotate := runTail(t, path, incremental.Pos, incremental.Inode, 1<<20, 0)
	if len(afterRotate.Lines) != 1 || afterRotate.Lines[0] != "fresh after rotation" {
		t.Fatalf("轮转后读取错误: %#v", afterRotate.Lines)
	}

	// 通配符：选中最近修改的文件
	if err := os.WriteFile(filepath.Join(dir, "worker.log"), []byte("worker line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	globbed := runTail(t, filepath.Join(dir, "*.log"), 0, 0, 1<<20, 0)
	if len(globbed.Lines) != 1 || globbed.Lines[0] != "worker line" {
		t.Fatalf("通配符选文件错误: %#v", globbed.Lines)
	}

	// 单轮上限：超出部分计入 dropped 且不阻塞位点推进
	capped, err := ParseTailOutput(execTail(t, path, 0, 0, 5, 0))
	if err != nil {
		t.Fatal(err)
	}
	if capped.Dropped <= 0 {
		t.Fatalf("未触发上限保护: %+v", capped)
	}
}

func execTail(t *testing.T, file string, pos, inode, maxBytes, backBytes int64) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "tail.sh")
	if err := os.WriteFile(script, []byte(TailScript()), 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", script, file, itoa(pos), itoa(inode), itoa(maxBytes), itoa(backBytes))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("tail.sh 执行失败: %v", err)
	}
	return string(out)
}

func runTail(t *testing.T, file string, pos, inode, maxBytes, backBytes int64) *TailResult {
	t.Helper()
	result, err := ParseTailOutput(execTail(t, file, pos, inode, maxBytes, backBytes))
	if err != nil {
		t.Fatalf("tail.sh 输出解析失败: %v", err)
	}
	return result
}

func itoa(v int64) string {
	return fmt.Sprintf("%d", v)
}

func TestParseLineTime(t *testing.T) {
	cases := map[string]string{
		"2026-10-15 08:12:33,456 ERROR boom":                         "2026-10-15 08:12:33",
		"2026-10-15T08:12:33+08:00 INFO started":                     "2026-10-15 08:12:33",
		`10.0.0.1 - - [15/Oct/2026:08:12:33 +0800] "GET / HTTP/1.1"`: "2026-10-15 08:12:33",
		"Oct 15 08:12:33 host app[1]: message":                       "2026-10-15 08:12:33",
		"plain text without time":                                    "",
	}
	for line, want := range cases {
		got := ParseLineTime(line)
		if want == "" {
			if !got.IsZero() {
				t.Errorf("不应解析出时间: %q -> %s", line, got)
			}
			continue
		}
		if got.IsZero() {
			t.Errorf("时间解析失败: %q", line)
			continue
		}
		if got.Local().Format("2006-01-02 15:04:05") != want {
			t.Errorf("时间不一致 line=%q got=%s want=%s", line, got.Format("2006-01-02 15:04:05"), want)
		}
	}
}

func TestDetectLevel(t *testing.T) {
	cases := map[string]string{
		"2026-10-15 08:12:33 ERROR boom": "ERROR",
		"[WARN] disk almost full":        "WARN",
		"INFO started server":            "INFO",
		"GET /health 200":                "",
	}
	for line, want := range cases {
		if got := DetectLevel(line, ""); got != want {
			t.Errorf("级别识别错误 line=%q got=%q want=%q", line, got, want)
		}
	}
	if got := DetectLevel("2026-10-15 level=SEVERE x", `level=(\w+)`); got != "ERROR" {
		t.Errorf("自定义正则提取失败: %s", got)
	}
}

func TestLabelSafety(t *testing.T) {
	if SafeLabelValue(`host"1`) != "" {
		t.Error("含引号的标签值应被拒绝")
	}
	if SafeLabelValue("10.0.0.1") != "10.0.0.1" {
		t.Error("正常标签值不应被拒绝")
	}
	selector := BuildSelector(map[string]string{"job": "ops_log", "host": "10.0.0.1"})
	if selector != `{host="10.0.0.1",job="ops_log"}` {
		t.Fatalf("流选择器生成错误: %s", selector)
	}
	if got := QuoteLabelValue(`say "hi"`); got != `say \"hi\"` {
		t.Errorf("引号未转义: %s", got)
	}
}

func TestLokiPushQueryRoundTrip(t *testing.T) {
	if os.Getenv("LOKI_E2E") == "" {
		t.Skip("需要本机 Loki，设置 LOKI_E2E=1 后运行")
	}
	t.Setenv("LOKI_URL", "http://localhost:3100")
	client := NewClient()
	ctx := context.Background()
	if !client.Available(ctx) {
		t.Skipf("本机 Loki 不可达: %s", client.BaseURL())
	}

	job := "ops_log_e2e"
	now := time.Now()
	labels := map[string]string{"job": job, "host": "127.0.0.1", "level": "ERROR"}
	line := fmt.Sprintf("e2e ERROR simulated failure %d", now.Unix())
	stream := Stream{Labels: labels, Entries: []LogEntry{{Line: line, Timestamp: now}}}
	if err := client.Push(ctx, []Stream{stream}); err != nil {
		t.Fatalf("写入 Loki 失败: %v", err)
	}

	selector := BuildSelector(map[string]string{"job": job})
	deadline := time.Now().Add(30 * time.Second)
	var lines []LogLine
	for time.Now().Before(deadline) {
		got, err := client.QueryRange(ctx, selector+` |= "ERROR"`, now.Add(-time.Minute), now.Add(2*time.Minute), 100, "backward")
		if err != nil {
			t.Fatalf("区间查询失败: %v", err)
		}
		if len(got) > 0 {
			lines = got
			break
		}
		time.Sleep(time.Second)
	}
	if len(lines) == 0 {
		t.Fatal("查询不到刚写入的日志，Loki 索引可能未就绪")
	}
	if lines[0].Labels["host"] != "127.0.0.1" {
		t.Fatalf("标签回传错误: %+v", lines[0].Labels)
	}

	points, err := client.QueryMetric(ctx, fmt.Sprintf("sum(count_over_time(%s [1m]))", selector), now.Add(-time.Minute), now.Add(time.Minute), "1m")
	if err != nil {
		t.Fatalf("直方图查询失败: %v", err)
	}
	var sum float64
	for _, p := range points {
		sum += p.Value
	}
	if sum < 1 {
		t.Fatalf("直方图计数为 0: %+v", points)
	}

	samples, err := client.QueryVector(ctx, fmt.Sprintf("sum by (host) (count_over_time(%s [1m]))", selector), now)
	if err != nil {
		t.Fatalf("即时向量查询失败: %v", err)
	}
	if len(samples) == 0 || samples[0].Value < 1 {
		t.Fatalf("按主机计数错误: %+v", samples)
	}

	names, err := client.Labels(ctx)
	if err != nil {
		t.Fatalf("标签列表查询失败: %v", err)
	}
	found := false
	for _, name := range names {
		if name == "job" {
			found = true
		}
	}
	if !found {
		t.Fatalf("标签列表缺少 job: %v", names)
	}

	values, err := client.LabelValues(ctx, "job", nil, nil)
	if err != nil {
		t.Fatalf("标签取值查询失败: %v", err)
	}
	found = false
	for _, value := range values {
		if value == job {
			found = true
		}
	}
	if !found {
		t.Fatalf("标签取值缺少 %s: %v", job, values)
	}
}
