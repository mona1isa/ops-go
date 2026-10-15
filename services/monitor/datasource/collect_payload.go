package datasource

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

//go:embed assets/collect.sh
var collectScript string

// CollectionScript 返回下发到目标主机执行的采集脚本
func CollectionScript() string {
	return collectScript
}

// DiskMetric 单个挂载点使用率
type DiskMetric struct {
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	UsedPercent float64 `json:"used_percent"`
}

// NetMetric 单个网卡收发速率
type NetMetric struct {
	Device        string  `json:"device"`
	RxBytesPerSec float64 `json:"rx_bytes_per_sec"`
	TxBytesPerSec float64 `json:"tx_bytes_per_sec"`
}

// CollectPayload 采集脚本输出的 JSON 结构
type CollectPayload struct {
	CPUPercent     float64      `json:"cpu_percent"`
	MemUsedPercent float64      `json:"mem_used_percent"`
	Load1          float64      `json:"load1"`
	UptimeSeconds  float64      `json:"uptime_seconds"`
	OS             string       `json:"os"`
	Disks          []DiskMetric `json:"disks"`
	Nets           []NetMetric  `json:"nets"`
}

// ParseCollectOutput 从 SSH 输出中解析采集结果
// 采集走 CombinedOutput，stderr 可能混入内容，因此截取最后一个完整 JSON 对象
func ParseCollectOutput(output string) (*CollectPayload, error) {
	end := strings.LastIndex(output, "}")
	if end < 0 {
		return nil, fmt.Errorf("采集输出中未找到 JSON: %.200s", output)
	}
	// 从末尾向前配对到本次输出的起始大括号
	depth := 0
	start := -1
	for i := end; i >= 0; i-- {
		switch output[i] {
		case '}':
			depth++
		case '{':
			depth--
			if depth == 0 {
				start = i
			}
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("采集输出 JSON 不完整: %.200s", output)
	}

	body := output[start : end+1]
	if strings.Contains(body, `"error"`) {
		return nil, fmt.Errorf("采集脚本执行失败: %s", strings.Trim(body, "{}\""))
	}

	var payload CollectPayload
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, fmt.Errorf("解析采集结果失败: %w", err)
	}
	if !isValidFloat(payload.CPUPercent) && !isValidFloat(payload.MemUsedPercent) {
		return nil, fmt.Errorf("采集结果缺少有效指标")
	}
	return &payload, nil
}

// 指标名与语义：采集脚本产出的是已计算的百分比 gauge，因此统一使用 ops_ 前缀
const (
	MetricCPUUsage  = "ops_cpu_usage_percent"
	MetricMemUsage  = "ops_mem_usage_percent"
	MetricLoad1     = "ops_load1"
	MetricUptime    = "ops_uptime_seconds"
	MetricDiskUsage = "ops_disk_usage_percent"
	MetricNetRx     = "ops_net_receive_bytes_per_second"
	MetricNetTx     = "ops_net_transmit_bytes_per_second"
	MetricHostInfo  = "ops_host_info"
)

// BuildExposition 构造 Prometheus 文本暴露格式
// job / instance 只出现在 Pushgateway 的 URL 分组里，正文中重复会被告拒收
func BuildExposition(p *CollectPayload) string {
	var b strings.Builder
	writeSample := func(name, help, labels string, value float64) {
		if !isValidFloat(value) {
			return
		}
		b.WriteString("# HELP " + name + " " + help + "\n")
		b.WriteString("# TYPE " + name + " gauge\n")
		if labels == "" {
			b.WriteString(name + " " + formatFloat(value) + "\n")
		} else {
			b.WriteString(name + "{" + labels + "} " + formatFloat(value) + "\n")
		}
	}

	writeSample(MetricCPUUsage, "CPU 使用率（百分比）", "", p.CPUPercent)
	writeSample(MetricMemUsage, "内存使用率（百分比）", "", p.MemUsedPercent)
	writeSample(MetricLoad1, "1 分钟平均负载", "", p.Load1)
	writeSample(MetricUptime, "系统运行时长（秒）", "", p.UptimeSeconds)

	// 主机操作系统信息，值恒为 1，仅用于展示与标签关联
	if p.OS != "" {
		b.WriteString("# HELP ops_host_info 主机基础信息，值恒为 1\n")
		b.WriteString("# TYPE ops_host_info gauge\n")
		b.WriteString(MetricHostInfo + `{os="` + escapeLabelValue(p.OS) + `"} 1` + "\n")
	}

	// 磁盘与网卡按维度展开，采集脚本已过滤伪文件系统和容器临时网卡
	diskHelpWritten := false
	for _, disk := range p.Disks {
		if !isValidFloat(disk.UsedPercent) {
			continue
		}
		if !diskHelpWritten {
			b.WriteString("# HELP " + MetricDiskUsage + " 磁盘使用率（百分比）\n")
			b.WriteString("# TYPE " + MetricDiskUsage + " gauge\n")
			diskHelpWritten = true
		}
		labels := `mountpoint="` + escapeLabelValue(disk.Mountpoint) +
			`",fstype="` + escapeLabelValue(disk.Fstype) + `"`
		b.WriteString(MetricDiskUsage + "{" + labels + "} " + formatFloat(disk.UsedPercent) + "\n")
	}

	netHelpWritten := false
	for _, net := range p.Nets {
		if net.Device == "" {
			continue
		}
		if !isValidFloat(net.RxBytesPerSec) && !isValidFloat(net.TxBytesPerSec) {
			continue
		}
		if !netHelpWritten {
			b.WriteString("# HELP " + MetricNetRx + " 网卡接收速率（字节/秒）\n")
			b.WriteString("# TYPE " + MetricNetRx + " gauge\n")
			b.WriteString("# HELP " + MetricNetTx + " 网卡发送速率（字节/秒）\n")
			b.WriteString("# TYPE " + MetricNetTx + " gauge\n")
			netHelpWritten = true
		}
		labels := `device="` + escapeLabelValue(net.Device) + `"`
		if isValidFloat(net.RxBytesPerSec) {
			b.WriteString(MetricNetRx + "{" + labels + "} " + formatFloat(net.RxBytesPerSec) + "\n")
		}
		if isValidFloat(net.TxBytesPerSec) {
			b.WriteString(MetricNetTx + "{" + labels + "} " + formatFloat(net.TxBytesPerSec) + "\n")
		}
	}

	return b.String()
}

// formatFloat 暴露格式要求纯数字，NaN/Inf 由调用方提前过滤
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func isValidFloat(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}

func escapeLabelValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	v = strings.ReplaceAll(v, "\n", `\n`)
	return v
}
