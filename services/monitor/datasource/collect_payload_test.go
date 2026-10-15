package datasource

import (
	"strconv"
	"strings"
	"testing"
)

func samplePayload() *CollectPayload {
	return &CollectPayload{
		CPUPercent:     12.5,
		MemUsedPercent: 41.3,
		Load1:          0.42,
		UptimeSeconds:  863400,
		OS:             "Linux",
		Disks: []DiskMetric{
			{Mountpoint: "/", Fstype: "ext4", UsedPercent: 63},
			{Mountpoint: "/data", Fstype: "xfs", UsedPercent: 88.5},
		},
		Nets: []NetMetric{
			{Device: "eth0", RxBytesPerSec: 152340, TxBytesPerSec: 88120},
		},
	}
}

func TestParseCollectOutput(t *testing.T) {
	out := `{"cpu_percent":12.50,"mem_used_percent":41.30,"load1":0.42,"uptime_seconds":863400,"os":"Linux","disks":[{"mountpoint":"/","fstype":"ext4","used_percent":63}],"nets":[{"device":"eth0","rx_bytes_per_sec":152340,"tx_bytes_per_sec":88120}]}`

	payload, err := ParseCollectOutput(out)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if payload.CPUPercent != 12.5 || payload.MemUsedPercent != 41.3 {
		t.Fatalf("指标解析不正确: %+v", payload)
	}
	if len(payload.Disks) != 1 || payload.Disks[0].Mountpoint != "/" {
		t.Fatalf("磁盘解析不正确: %+v", payload.Disks)
	}
	if len(payload.Nets) != 1 || payload.Nets[0].RxBytesPerSec != 152340 {
		t.Fatalf("网卡解析不正确: %+v", payload.Nets)
	}
}

// SSH 侧使用 CombinedOutput，脚本自身的告警信息可能混在 JSON 前面
func TestParseCollectOutputWithNoise(t *testing.T) {
	out := "df: /old/mnt: Stale file handle\n" +
		`{"cpu_percent":1,"mem_used_percent":2,"load1":0,"uptime_seconds":10,"os":"Linux","disks":[],"nets":[]}` + "\n"

	payload, err := ParseCollectOutput(out)
	if err != nil {
		t.Fatalf("含噪输出应可解析: %v", err)
	}
	if payload.CPUPercent != 1 {
		t.Fatalf("解析结果异常: %+v", payload)
	}
}

func TestParseCollectOutputErrors(t *testing.T) {
	cases := map[string]string{
		"无 JSON":   "connection timeout",
		"脚本错误":     `{"error":"unsupported_platform"}`,
		"JSON 不完整": `{"cpu_percent":1,"disks":[`,
	}
	for name, out := range cases {
		if _, err := ParseCollectOutput(out); err == nil {
			t.Fatalf("%s 应当解析失败", name)
		}
	}
}

func TestBuildExposition(t *testing.T) {
	text := BuildExposition(samplePayload())

	for _, want := range []string{
		"# TYPE ops_cpu_usage_percent gauge",
		"ops_cpu_usage_percent 12.5",
		"ops_mem_usage_percent 41.3",
		`ops_disk_usage_percent{mountpoint="/",fstype="ext4"} 63`,
		`ops_disk_usage_percent{mountpoint="/data",fstype="xfs"} 88.5`,
		`ops_net_receive_bytes_per_second{device="eth0"} 152340`,
		`ops_net_transmit_bytes_per_second{device="eth0"} 88120`,
		`ops_host_info{os="Linux"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("暴露文本缺少 %q\n实际内容:\n%s", want, text)
		}
	}

	// job / instance 只能出现在 Pushgateway 的 URL 分组里，正文重复会被拒收
	if strings.Contains(text, `instance=`) || strings.Contains(text, `job=`) {
		t.Fatalf("暴露文本不应包含 job/instance 标签:\n%s", text)
	}
}

// 每行样本必须以可解析的浮点值结尾。
// 曾因 `\n` 写反引号原始字符串里而输出字面量 \n，导致 Pushgateway 报
// "expected float as value"，整批指标被拒收，这里做格式回归防护。
func TestBuildExpositionLineFormat(t *testing.T) {
	text := BuildExposition(samplePayload())
	if strings.Contains(text, "\\n") {
		t.Fatalf("暴露文本含字面量 \\n，换行未生效:\n%s", text)
	}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	samples := 0
	for i, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		value := fields[len(fields)-1]
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			t.Fatalf("第 %d 行不是合法样本，结尾值=%q 行=%q", i+1, value, line)
		}
		samples++
	}
	if samples == 0 {
		t.Fatal("暴露文本没有任何样本行")
	}
}

func TestBuildExpositionSkipsInvalidValues(t *testing.T) {
	payload := &CollectPayload{
		CPUPercent:     1,
		MemUsedPercent: 0,
		Disks:          []DiskMetric{{Mountpoint: "bad\"", Fstype: "ext4", UsedPercent: 50}},
	}
	text := BuildExposition(payload)
	if strings.Contains(text, "NaN") || strings.Contains(text, "+Inf") {
		t.Fatalf("非法数值应被过滤:\n%s", text)
	}
	// 标签值中的双引号需要转义，否则破坏暴露格式
	if !strings.Contains(text, `mountpoint="bad\""`) {
		t.Fatalf("标签值未正确转义:\n%s", text)
	}
}

func TestCollectionScriptIsEmbedded(t *testing.T) {
	script := CollectionScript()
	if !strings.Contains(script, "cpu_percent") || !strings.Contains(script, "/proc/stat") {
		t.Fatalf("采集脚本未正确嵌入，长度=%d", len(script))
	}
	// 脚本以 heredoc 下发，不能与结束标记冲突
	if strings.Contains(script, "OPS_SCRIPT_EOF") {
		t.Fatal("采集脚本包含 heredoc 结束标记，会导致下发失败")
	}
}
