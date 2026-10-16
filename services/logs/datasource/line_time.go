package datasource

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Loki 只接受近实时的事件时间，过旧的时间戳会被拒收（超出保留期）
const minValidTimeYear = 2000

var (
	// ISO / 常见 "年-月-日 时:分:秒" 前缀
	headTimeRe = regexp.MustCompile(`^[\[{]?(\d{4})[-/](\d{2})[-/](\d{2})[T ](\d{2}):(\d{2}):(\d{2})`)
	// nginx common log: [16/Oct/2026:12:00:00 +0800]
	bracketTimeRe = regexp.MustCompile(`\[(\d{2}/[A-Za-z]{3}/\d{4}:\d{2}:\d{2}:\d{2} [+\-]\d{4})\]`)
	// syslog: Oct 16 12:00:00
	syslogTimeRe = regexp.MustCompile(`^([A-Z][a-z]{2}) +(\d{1,2}) (\d{2}):(\d{2}):(\d{2})`)
)

// ParseLineTime 从日志行提取事件时间，识别失败返回零值，调用方按采集时间落库。
// 只扫描行首有限长度，避免对超长行做全文正则匹配。
func ParseLineTime(line string) time.Time {
	head := line
	if len(head) > 64 {
		head = head[:64]
	}

	if m := headTimeRe.FindStringSubmatch(head); len(m) == 7 {
		year := toInt(m[1])
		if year < minValidTimeYear {
			return time.Time{}
		}
		return time.Date(year, time.Month(toInt(m[2])), toInt(m[3]),
			toInt(m[4]), toInt(m[5]), toInt(m[6]), 0, time.Local)
	}

	if m := bracketTimeRe.FindStringSubmatch(head); len(m) > 1 {
		if parsed, err := time.Parse("02/Jan/2006:15:04:05 -0700", m[1]); err == nil {
			return parsed
		}
	}

	// syslog 不带年份，按当前年份就近推断（未来月份视为去年）
	if m := syslogTimeRe.FindStringSubmatch(head); len(m) == 6 {
		month := monthOf(m[1])
		if month == 0 {
			return time.Time{}
		}
		day, hour, minute, sec := toInt(m[2]), toInt(m[3]), toInt(m[4]), toInt(m[5])
		now := time.Now()
		parsed := time.Date(now.Year(), month, day, hour, minute, sec, 0, time.Local)
		if parsed.After(now.AddDate(0, 1, 0)) {
			parsed = parsed.AddDate(-1, 0, 0)
		}
		return parsed
	}

	return time.Time{}
}

var monthNames = map[string]time.Month{
	"Jan": time.January, "Feb": time.February, "Mar": time.March, "Apr": time.April,
	"May": time.May, "Jun": time.June, "Jul": time.July, "Aug": time.August,
	"Sep": time.September, "Oct": time.October, "Nov": time.November, "Dec": time.December,
}

// monthOf 解析 syslog 的月份缩写，无法识别返回 0
func monthOf(name string) time.Month {
	if month, ok := monthNames[strings.TrimSpace(name)]; ok {
		return month
	}
	return 0
}

func toInt(value string) int {
	num, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return num
}
