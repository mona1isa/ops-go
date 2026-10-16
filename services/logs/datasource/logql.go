package datasource

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// MaxLabelValue 标签取值长度上限，防止异常内容把 Loki 索引撑爆
const MaxLabelValue = 128

// SafeLabelValue 规范化标签取值：含控制字符/引号/反斜杠、非 ASCII 或超长时返回空串（该标签不下发）。
// 标签取值全部走参数化 JSON 下发，此处过滤是为了控制 Loki 索引基数与内容安全。
func SafeLabelValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > MaxLabelValue {
		return ""
	}
	for i := 0; i < len(value); i++ {
		char := value[i]
		if char < 0x20 || char == 0x7f || char >= 0x80 {
			return ""
		}
		// 空格保留（路径可能含空格），但引号与反斜杠会干扰 LogQL 可读性
		if char == '"' || char == '\\' {
			return ""
		}
	}
	return value
}

// QuoteLabelValue LogQL 字符串里的取值需转义反斜杠和双引号
func QuoteLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

// BuildSelector 用标签集合拼出 LogQL 流选择器，如 {job="ops_log",host="10.0.0.1"}
func BuildSelector(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := SafeLabelValue(labels[key])
		if value == "" {
			continue
		}
		parts = append(parts, key+"=\""+QuoteLabelValue(value)+"\"")
	}
	if len(parts) == 0 {
		return ""
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// builtinLevels 内置级别识别关键字，按严重程度从高到低匹配
var builtinLevels = []string{"FATAL", "ERROR", "WARN", "WARNING", "INFO", "DEBUG", "TRACE"}

// levelRegexpCache 级别正则缓存，避免逐行重复编译
var levelRegexpCache sync.Map

func compileLevelRegexp(pattern string) *regexp.Regexp {
	if cached, ok := levelRegexpCache.Load(pattern); ok {
		if re, typed := cached.(*regexp.Regexp); typed {
			return re
		}
	}
	compiled, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		compiled = nil
	}
	levelRegexpCache.Store(pattern, compiled)
	return compiled
}

// DetectLevel 识别单行日志级别。custom 为采集源配置的正则（需含捕获组），为空时用内置关键字。
// 识别失败返回空串，调用方按「无级别」处理。
func DetectLevel(line string, custom string) string {
	if strings.TrimSpace(custom) != "" {
		re := compileLevelRegexp(custom)
		if re == nil {
			return ""
		}
		if groups := re.FindStringSubmatch(line); len(groups) > 1 {
			return normalizeLevel(groups[1])
		}
		return ""
	}

	upper := strings.ToUpper(line)
	for _, level := range builtinLevels {
		if strings.Contains(upper, level) {
			if level == "WARNING" {
				return "WARN"
			}
			return level
		}
	}
	return ""
}

func normalizeLevel(value string) string {
	level := strings.ToUpper(strings.TrimSpace(value))
	switch level {
	case "WARNING":
		return "WARN"
	case "ERR", "SEVERE":
		return "ERROR"
	case "CRITICAL", "FATAL", "EMERG", "ALERT":
		return "FATAL"
	case "NOTICE", "INFO":
		return "INFO"
	case "DEBUG", "TRACE":
		return level
	case "WARN":
		return "WARN"
	case "ERROR":
		return "ERROR"
	}
	return ""
}

// ParseDurationSec 把 "5m"/"30s"/"1h" 解析为秒，失败返回 0
func ParseDurationSec(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	// Loki 写法允许不带单位后缀的秒数
	if num, err := strconv.Atoi(value); err == nil && num > 0 {
		return num
	}
	unit := value[len(value)-1]
	digits := strings.TrimRight(value, string(unit))
	num, err := strconv.Atoi(digits)
	if err != nil || num <= 0 {
		return 0
	}
	switch unit {
	case 's':
		return num
	case 'm':
		return num * 60
	case 'h':
		return num * 3600
	case 'd':
		return num * 86400
	}
	return 0
}
