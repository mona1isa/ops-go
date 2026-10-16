package logs

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zhany/ops-go/controllers/logs/api"
	"github.com/zhany/ops-go/services/logs/datasource"
)

// 检索参数上限，避免单次查询把 Loki 打爆
const (
	maxQueryLimit     = 5000
	defaultQueryLimit = 500
	maxQueryWindow    = 24 * time.Hour
	defaultWindow     = time.Hour
)

// QueryService 日志检索服务：把界面条件翻译成 LogQL 并访问 Loki
type QueryService struct {
	client *datasource.Client
}

// NewQueryService 创建检索服务
func NewQueryService() *QueryService {
	return &QueryService{client: datasource.NewClient()}
}

// Status 数据源可达性，供界面提示「日志服务未就绪」
func (s *QueryService) Status(ctx context.Context) map[string]any {
	available := s.client.Available(ctx)
	return map[string]any{
		"available": available,
		"url":       s.client.BaseURL(),
	}
}

// BuildLogQL 由查询条件拼出 LogQL 检索表达式
func (s *QueryService) BuildLogQL(request api.LogQueryRequest) (string, error) {
	selector := strings.TrimSpace(request.Selector)
	if selector == "" {
		labels := map[string]string{"job": request.Job}
		if request.Host != "" {
			labels["host"] = request.Host
		}
		if request.Source != "" {
			labels["source"] = request.Source
		}
		if request.Level != "" {
			labels["level"] = strings.ToUpper(request.Level)
		}
		selector = datasource.BuildSelector(labels)
	}
	if selector == "" {
		return "", errors.New("请至少选择一个日志来源（job）或填写流选择器")
	}
	if err := validateSelector(selector); err != nil {
		return "", err
	}

	builder := strings.Builder{}
	builder.WriteString(selector)

	for _, keyword := range splitKeywords(request.Keyword) {
		builder.WriteString(` |= "` + datasource.QuoteLabelValue(keyword) + `"`)
	}
	for _, keyword := range splitKeywords(request.Exclude) {
		builder.WriteString(` != "` + datasource.QuoteLabelValue(keyword) + `"`)
	}

	filter := strings.TrimSpace(request.LineFilter)
	if filter != "" {
		if err := validateLineFilter(filter); err != nil {
			return "", err
		}
		builder.WriteString(" " + filter)
	}
	return builder.String(), nil
}

// Query 检索日志明细
func (s *QueryService) Query(ctx context.Context, request api.LogQueryRequest) ([]datasource.LogLine, error) {
	logql, err := s.BuildLogQL(request)
	if err != nil {
		return nil, err
	}
	start, end, err := resolveWindow(request.StartAt, request.EndAt)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(request.Limit)
	return s.client.QueryRange(ctx, logql, start, end, limit, request.Direction)
}

// Histogram 时间分布直方图：用 count_over_time 指标查询按步长聚合条数
func (s *QueryService) Histogram(ctx context.Context, request api.LogHistogramRequest) ([]datasource.HistogramPoint, error) {
	logql, err := s.BuildLogQL(request.LogQueryRequest)
	if err != nil {
		return nil, err
	}
	start, end, err := resolveWindow(request.StartAt, request.EndAt)
	if err != nil {
		return nil, err
	}

	step := normalizeStep(request.Step, end.Sub(start))
	expr := fmt.Sprintf("sum(count_over_time(%s [%s]))", logql, step)
	return s.client.QueryMetric(ctx, expr, start, end.Add(time.Second), step)
}

// Context 命中日志的前后 N 行。
// Loki 没有原生上下文接口，按同一标签流在命中时刻前后各取一小段窗口即可。
func (s *QueryService) Context(ctx context.Context, request api.LogContextRequest) (map[string][]datasource.LogLine, error) {
	selector := strings.TrimSpace(request.Selector)
	if selector == "" {
		labels := map[string]string{"job": request.Job}
		if request.Host != "" {
			labels["host"] = request.Host
		}
		if request.Source != "" {
			labels["source"] = request.Source
		}
		if request.Path != "" {
			labels["path"] = request.Path
		}
		selector = datasource.BuildSelector(labels)
	}
	if selector == "" {
		return nil, errors.New("缺少标签流信息，无法定位上下文")
	}
	if err := validateSelector(selector); err != nil {
		return nil, err
	}

	center, err := parseTimestamp(request.Timestamp)
	if err != nil {
		return nil, err
	}
	lines := request.Lines
	if lines <= 0 {
		lines = 20
	}
	if lines > 200 {
		lines = 200
	}

	// 前后各扩 5 分钟窗口，同一秒内多条日志按行序截断
	span := time.Duration(lines) * time.Second
	beforeStart := center.Add(-5*time.Minute - span)
	afterEnd := center.Add(5*time.Minute + span)

	before, err := s.client.QueryRange(ctx, selector, beforeStart, center, lines, "backward")
	if err != nil {
		return nil, err
	}
	after, err := s.client.QueryRange(ctx, selector, center.Add(time.Nanosecond), afterEnd, lines, "forward")
	if err != nil {
		return nil, err
	}

	return map[string][]datasource.LogLine{
		"before": reverse(before),
		"after":  after,
	}, nil
}

// Labels 全部标签名
func (s *QueryService) Labels(ctx context.Context) ([]string, error) {
	return s.client.Labels(ctx)
}

// LabelValues 指定标签的取值，默认回溯 7 天，避免列出历史残留值
func (s *QueryService) LabelValues(ctx context.Context, label string) ([]string, error) {
	name := datasource.SafeLabelValue(label)
	if name == "" || !validLabelName(name) {
		return nil, errors.New("标签名非法")
	}
	start := time.Now().AddDate(0, 0, -7)
	end := time.Now()
	return s.client.LabelValues(ctx, name, &start, &end)
}

// Sources 已入库的采集来源（host 标签取值），供筛选下拉
func (s *QueryService) Sources(ctx context.Context) ([]string, error) {
	start := time.Now().AddDate(0, 0, -7)
	end := time.Now()
	return s.client.LabelValues(ctx, "host", &start, &end)
}

// resolveWindow 解析时间范围，缺省取最近 1 小时，最长 24 小时
func resolveWindow(startAt, endAt string) (time.Time, time.Time, error) {
	end := time.Now()
	start := end.Add(-defaultWindow)

	if trimmed := strings.TrimSpace(startAt); trimmed != "" {
		parsed, err := parseRangeTime(trimmed)
		if err != nil {
			return start, end, err
		}
		start = parsed
	}
	if trimmed := strings.TrimSpace(endAt); trimmed != "" {
		parsed, err := parseRangeTime(trimmed)
		if err != nil {
			return start, end, err
		}
		end = parsed
	}
	if !end.After(start) {
		return start, end, errors.New("结束时间必须晚于开始时间")
	}
	if end.Sub(start) > maxQueryWindow {
		return start, end, fmt.Errorf("单次查询时间跨度不能超过 %d 小时", int(maxQueryWindow/time.Hour))
	}
	return start, end, nil
}

func parseRangeTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	// 前端也可能直接传毫秒时间戳
	if num, err := strconv.ParseInt(value, 10, 64); err == nil && num > 0 {
		if num > 1e12 {
			return time.UnixMilli(num), nil
		}
		return time.Unix(num, 0), nil
	}
	return time.Time{}, fmt.Errorf("时间格式无法识别: %s", value)
}

func parseTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("缺少命中日志的时间戳")
	}
	if num, err := strconv.ParseInt(value, 10, 64); err == nil && num > 0 {
		return time.Unix(0, num), nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("时间戳格式无法识别: %s", value)
	}
	return parsed, nil
}

func normalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultQueryLimit
	case limit > maxQueryLimit:
		return maxQueryLimit
	}
	return limit
}

// normalizeStep 直方图步长：显式指定则校验，否则按时间跨度自动取 30~60 个桶
func normalizeStep(step string, window time.Duration) string {
	value := strings.TrimSpace(step)
	if value != "" {
		seconds := datasource.ParseDurationSec(value)
		if seconds < 10 || seconds > 3600 {
			return ""
		}
		return value
	}
	target := int(window/time.Minute)/50 + 1
	switch {
	case target <= 1:
		return "10s"
	case target <= 10:
		return "1m"
	case target <= 30:
		return "5m"
	case target <= 120:
		return "15m"
	}
	return "1h"
}

// splitKeywords 空格分隔的多关键字，过滤掉纯空白项
func splitKeywords(value string) []string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return nil
	}
	return fields
}

var selectorPattern = regexp.MustCompile(`^\{[^{}\n]*\}$`)

// validateSelector 只允许标准的 {label="value",...} 形式，禁止嵌套括号与换行注入
func validateSelector(selector string) error {
	if len(selector) > 512 {
		return errors.New("流选择器过长")
	}
	if !selectorPattern.MatchString(selector) {
		return errors.New("流选择器格式非法，应形如 {job=\"ops_log\"}")
	}
	if strings.Count(selector, "\"")%2 != 0 {
		return errors.New("流选择器引号不匹配")
	}
	return nil
}

// validateLineFilter 行过滤器必须是成对引号的字面量或正则片段
func validateLineFilter(filter string) error {
	if len(filter) > 256 {
		return errors.New("行过滤器过长")
	}
	if strings.ContainsAny(filter, "\n\r`") {
		return errors.New("行过滤器不允许换行与反引号")
	}
	if !strings.HasPrefix(filter, "|") {
		return errors.New("行过滤器需以 | 开头，如 |~ \"error|fatal\"")
	}
	if strings.Count(filter, "\"")%2 != 0 {
		return errors.New("行过滤器引号不匹配")
	}
	return nil
}

func reverse(lines []datasource.LogLine) []datasource.LogLine {
	out := make([]datasource.LogLine, len(lines))
	for i, line := range lines {
		out[len(lines)-1-i] = line
	}
	return out
}
