package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrLokiUnavailable Loki 未配置或不可达
var ErrLokiUnavailable = errors.New("日志数据源不可用")

// maxResponseBytes 单次检索响应体上限，防止极端查询打爆内存
const maxResponseBytes = 64 << 20

// LogEntry 一条待写入的日志
type LogEntry struct {
	Line      string
	Timestamp time.Time
}

// Stream 一个标签流下的日志批次
type Stream struct {
	Labels  map[string]string
	Entries []LogEntry
}

// LogLine 检索返回的一行日志
type LogLine struct {
	Timestamp time.Time         `json:"timestamp"`
	Line      string            `json:"line"`
	Labels    map[string]string `json:"labels"`
}

// HistogramPoint 时间分布直方图的一个桶
type HistogramPoint struct {
	Timestamp int64   `json:"timestamp"` // Unix 秒
	Value     float64 `json:"value"`
}

// VectorSample 即时向量查询的一条序列样本
type VectorSample struct {
	Labels    map[string]string `json:"labels"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
}

// Client Loki HTTP 客户端：写入用 push，检索用 LogQL
type Client struct {
	base   string
	token  string
	client *http.Client

	mu      sync.RWMutex
	ok      bool
	checked time.Time
}

// NewClient 按环境变量构造 Loki 客户端
func NewClient() *Client {
	return &Client{base: LokiURL(), token: lokiToken(), client: newLokiHTTPClient(lokiTimeout())}
}

// BaseURL 数据源地址，供状态接口展示
func (c *Client) BaseURL() string { return c.base }

// Push 批量写入日志流。Loki 对成功请求返回 204。
// 行内时间超出 Loki 接受窗口时会被整批拒绝，此时把越界行的时间钳制到窗口边界后重推一次，
// 否则采集端会因写失败回退位点，每轮重复拉取同一段历史日志而永远无法推进。
func (c *Client) Push(ctx context.Context, streams []Stream) error {
	list := encodeStreams(streams)
	if len(list) == 0 {
		return nil
	}

	status, respBody, err := c.sendPush(ctx, list)
	if err != nil {
		return err
	}
	if status/100 == 2 {
		return nil
	}

	lower, upper := parseTimestampWindow(string(respBody))
	if lower.IsZero() && upper.IsZero() {
		c.setState(false)
		return fmt.Errorf("写入日志失败 status=%d body=%.300s", status, respBody)
	}
	clamped := clampStreams(streams, lower, upper)
	retryStatus, retryBody, retryErr := c.sendPush(ctx, encodeStreams(clamped))
	if retryErr != nil {
		return retryErr
	}
	if retryStatus/100 != 2 {
		c.setState(false)
		return fmt.Errorf("写入日志失败 status=%d body=%.300s", retryStatus, retryBody)
	}
	c.setState(true)
	return nil
}

// encodeStreams 转成 Loki push 请求体中的 streams 数组
func encodeStreams(streams []Stream) []map[string]any {
	list := make([]map[string]any, 0, len(streams))
	for _, stream := range streams {
		if len(stream.Entries) == 0 {
			continue
		}
		values := make([][]string, 0, len(stream.Entries))
		for _, entry := range stream.Entries {
			ts := entry.Timestamp
			if ts.IsZero() {
				ts = time.Now()
			}
			values = append(values, []string{strconv.FormatInt(ts.UnixNano(), 10), entry.Line})
		}
		list = append(list, map[string]any{"stream": stream.Labels, "values": values})
	}
	return list
}

// sendPush 发送一次写入请求，返回状态码与截断后的响应体
func (c *Client) sendPush(ctx context.Context, list []map[string]any) (int, []byte, error) {
	body, err := json.Marshal(map[string]any{"streams": list})
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/loki/api/v1/push", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.setState(false)
		return 0, nil, fmt.Errorf("%w: %v", ErrLokiUnavailable, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	return resp.StatusCode, respBody, nil
}

var (
	oldestAcceptablePattern = regexp.MustCompile(`oldest acceptable timestamp is: (\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z?)`)
	newestAcceptablePattern = regexp.MustCompile(`newest acceptable timestamp is: (\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z?)`)
)

// parseTimestampWindow 从 Loki 拒绝原因里解析出可接受的时间区间，缺失的一端为零值
func parseTimestampWindow(body string) (lower, upper time.Time) {
	if match := oldestAcceptablePattern.FindStringSubmatch(body); len(match) == 2 {
		if ts, err := time.Parse(time.RFC3339, match[1]); err == nil {
			lower = ts.Add(time.Second)
		}
	}
	if match := newestAcceptablePattern.FindStringSubmatch(body); len(match) == 2 {
		if ts, err := time.Parse(time.RFC3339, match[1]); err == nil {
			upper = ts.Add(-time.Second)
		}
	}
	return lower, upper
}

// clampStreams 把越界时间钳制进接受区间，日志内容保持原样（行内原始时间仍在正文里）
func clampStreams(streams []Stream, lower, upper time.Time) []Stream {
	clamped := make([]Stream, 0, len(streams))
	for _, stream := range streams {
		item := Stream{Labels: stream.Labels, Entries: make([]LogEntry, 0, len(stream.Entries))}
		for _, entry := range stream.Entries {
			ts := entry.Timestamp
			if ts.IsZero() {
				ts = time.Now()
			}
			if !lower.IsZero() && ts.Before(lower) {
				ts = lower
			}
			if !upper.IsZero() && ts.After(upper) {
				ts = upper
			}
			entry.Timestamp = ts
			item.Entries = append(item.Entries, entry)
		}
		clamped = append(clamped, item)
	}
	return clamped
}

// QueryRange 检索日志。direction 为 forward 时按时间正序返回。
func (c *Client) QueryRange(ctx context.Context, logql string, start, end time.Time, limit int, direction string) ([]LogLine, error) {
	params := url.Values{}
	params.Set("query", logql)
	params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	params.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if direction == "" {
		direction = "backward"
	}
	params.Set("direction", direction)

	raw, err := c.get(ctx, "/loki/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}
	var envelope apiResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("解析日志检索结果失败: %w", err)
	}
	if envelope.Status != "success" {
		return nil, fmt.Errorf("日志检索失败: %s", envelope.Error)
	}

	lines := make([]LogLine, 0)
	for _, stream := range envelope.streams() {
		for _, pair := range stream.Values {
			if len(pair) != 2 {
				continue
			}
			ts, err := parseNano(pair[0])
			if err != nil {
				continue
			}
			lines = append(lines, LogLine{Timestamp: ts, Line: pair[1], Labels: stream.Stream})
		}
	}
	sortLines(lines, direction == "forward")
	return lines, nil
}

// QueryMetric 执行 LogQL 指标查询（如 count_over_time），返回时间序列点，用于直方图与日志告警
func (c *Client) QueryMetric(ctx context.Context, logql string, start, end time.Time, step string) ([]HistogramPoint, error) {
	params := url.Values{}
	params.Set("query", logql)
	params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	params.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	if step != "" {
		params.Set("step", step)
	}

	raw, err := c.get(ctx, "/loki/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}
	var envelope apiResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("解析日志指标结果失败: %w", err)
	}
	if envelope.Status != "success" {
		return nil, fmt.Errorf("日志指标查询失败: %s", envelope.Error)
	}

	points := make([]HistogramPoint, 0)
	for _, series := range envelope.matrix() {
		for _, pair := range series.Values {
			value, err := strconv.ParseFloat(pair.Value, 64)
			if err != nil {
				continue
			}
			points = append(points, HistogramPoint{Timestamp: pair.Timestamp.Unix(), Value: value})
		}
	}
	// 多条序列合并后按时间升序，前端直方图直接可用
	sortPoints(points)
	return points, nil
}

// QueryVector 即时向量查询，返回每条序列的最新取值（日志告警按主机分组判定用）
func (c *Client) QueryVector(ctx context.Context, logql string, at time.Time) ([]VectorSample, error) {
	params := url.Values{}
	params.Set("query", logql)
	if !at.IsZero() {
		params.Set("time", strconv.FormatInt(at.UnixNano(), 10))
	}

	raw, err := c.get(ctx, "/loki/api/v1/query", params)
	if err != nil {
		return nil, err
	}
	var envelope apiResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("解析日志指标结果失败: %w", err)
	}
	if envelope.Status != "success" {
		return nil, fmt.Errorf("日志指标查询失败: %s", envelope.Error)
	}

	samples := make([]VectorSample, 0, len(envelope.vector()))
	for _, item := range envelope.vector() {
		value, err := strconv.ParseFloat(item.Value.Value, 64)
		if err != nil {
			continue
		}
		samples = append(samples, VectorSample{Labels: item.Metric, Value: value, Timestamp: item.Value.Timestamp})
	}
	return samples, nil
}

// QueryValue 即时指标查询，返回最新数值（日志告警汇总判定用）
func (c *Client) QueryValue(ctx context.Context, logql string, at time.Time) (float64, map[string]string, error) {
	samples, err := c.QueryVector(ctx, logql, at)
	if err != nil {
		return 0, nil, err
	}
	if len(samples) == 0 {
		return 0, nil, nil
	}
	return samples[0].Value, samples[0].Labels, nil
}

// Labels 返回全部标签名
func (c *Client) Labels(ctx context.Context) ([]string, error) {
	raw, err := c.get(ctx, "/loki/api/v1/labels", url.Values{})
	if err != nil {
		return nil, err
	}
	var out listResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("解析标签列表失败: %w", err)
	}
	return out.Data, nil
}

// LabelValues 返回指定标签的取值
func (c *Client) LabelValues(ctx context.Context, label string, start, end *time.Time) ([]string, error) {
	path := "/loki/api/v1/label/" + url.PathEscape(label) + "/values"
	params := url.Values{}
	if start != nil {
		params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	}
	if end != nil {
		params.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	}
	raw, err := c.get(ctx, path, params)
	if err != nil {
		return nil, err
	}
	var out listResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("解析标签取值失败: %w", err)
	}
	return out.Data, nil
}

// Available 探测 Loki 是否可达，结果缓存 30 秒避免逐台主机重试
func (c *Client) Available(ctx context.Context) bool {
	c.mu.RLock()
	if time.Since(c.checked) < 30*time.Second {
		ok := c.ok
		c.mu.RUnlock()
		return ok
	}
	c.mu.RUnlock()

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.base+"/ready", nil)
	if err != nil {
		return false
	}
	c.authorize(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.setState(false)
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 256))
	c.setState(resp.StatusCode == http.StatusOK)
	return c.ok
}

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	full := c.base + path
	if len(params) > 0 {
		full += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.setState(false)
		return nil, fmt.Errorf("%w: %v", ErrLokiUnavailable, err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if resp.StatusCode/100 != 2 {
		c.setState(false)
		return nil, fmt.Errorf("日志数据源返回异常 status=%d body=%.300s", resp.StatusCode, body)
	}
	if readErr != nil {
		return nil, fmt.Errorf("读取日志数据源响应失败: %w", readErr)
	}
	c.setState(true)
	return body, nil
}

func (c *Client) authorize(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c *Client) setState(ok bool) {
	c.mu.Lock()
	c.ok = ok
	c.checked = time.Now()
	c.mu.Unlock()
}

type apiResponse struct {
	Status string  `json:"status"`
	Error  string  `json:"error"`
	Data   apiData `json:"data"`
}

type apiData struct {
	ResultType string `json:"resultType"`
	raw        json.RawMessage
}

// UnmarshalJSON result 字段随 resultType 变化（streams/matrix/vector），先原样保留再按需解码
func (d *apiData) UnmarshalJSON(b []byte) error {
	var shell struct {
		ResultType string          `json:"resultType"`
		Result     json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(b, &shell); err != nil {
		return err
	}
	d.ResultType = shell.ResultType
	d.raw = shell.Result
	return nil
}

func (r apiResponse) streams() []streamData {
	var out []streamData
	json.Unmarshal(r.Data.raw, &out)
	return out
}

func (r apiResponse) matrix() []matrixData {
	var out []matrixData
	json.Unmarshal(r.Data.raw, &out)
	return out
}

func (r apiResponse) vector() []vectorData {
	var out []vectorData
	json.Unmarshal(r.Data.raw, &out)
	return out
}

type streamData struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type matrixData struct {
	Metric map[string]string `json:"metric"`
	Values []samplePair      `json:"values"`
}

type vectorData struct {
	Metric map[string]string `json:"metric"`
	Value  samplePair        `json:"value"`
}

// samplePair 指标采样点。注意与日志流不同：Loki/Prometheus 返回的时间是「秒」，
// 且为 JSON 数字（可带小数），取值是字符串，因此不能按 [][]string 解码。
type samplePair struct {
	Timestamp time.Time
	Value     string
}

func (p *samplePair) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) != 2 {
		return errors.New("指标采样点格式非法")
	}
	var seconds float64
	if err := json.Unmarshal(raw[0], &seconds); err != nil {
		return err
	}
	var value string
	if err := json.Unmarshal(raw[1], &value); err != nil {
		return err
	}
	whole := math.Floor(seconds)
	p.Timestamp = time.Unix(int64(whole), int64(math.Round((seconds-whole)*1e9)))
	p.Value = value
	return nil
}

type listResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

func parseNano(value string) (time.Time, error) {
	nano, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, nano).In(time.Local), nil
}

// sortLines 按时间排序，检索默认倒序（最新在前），forward 时正序
func sortLines(lines []LogLine, ascending bool) {
	sort.SliceStable(lines, func(i, j int) bool {
		if ascending {
			return lines[i].Timestamp.Before(lines[j].Timestamp)
		}
		return lines[i].Timestamp.After(lines[j].Timestamp)
	})
}

func sortPoints(points []HistogramPoint) {
	sort.SliceStable(points, func(i, j int) bool {
		return points[i].Timestamp < points[j].Timestamp
	})
}
