package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Sample 瞬时查询返回的一个时间序列样本
type Sample struct {
	Labels    map[string]string
	Value     float64
	Timestamp int64
}

// Instance 样本所属主机
func (s Sample) Instance() string {
	return s.Labels["instance"]
}

// Point 区间查询的一个数据点
type Point struct {
	Timestamp int64
	Value     float64
}

// Series 区间查询返回的一条时间序列
type Series struct {
	Labels map[string]string
	Points []Point
}

// PrometheusClient Prometheus HTTP API 客户端（只读查询）
type PrometheusClient struct {
	base    string
	token   string
	client  *http.Client
	mu      sync.RWMutex
	ready   bool
	checked time.Time
}

// NewPrometheusClient 按环境变量构造查询客户端
func NewPrometheusClient() *PrometheusClient {
	return &PrometheusClient{
		base:   PrometheusURL(),
		token:  prometheusToken(),
		client: newHTTPClient(15 * time.Second),
	}
}

type apiResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType string          `json:"errorType"`
	Error     string          `json:"error"`
}

type apiData struct {
	ResultType string            `json:"resultType"`
	Result     []json.RawMessage `json:"result"`
}

type vectorResult struct {
	Metric map[string]string  `json:"metric"`
	Value  [2]json.RawMessage `json:"value"`
}

type matrixResult struct {
	Metric map[string]string    `json:"metric"`
	Values [][2]json.RawMessage `json:"values"`
}

// Available 探测数据源是否可达，结果缓存 30 秒
func (c *PrometheusClient) Available(ctx context.Context) bool {
	c.mu.RLock()
	if time.Since(c.checked) < 30*time.Second {
		ready := c.ready
		c.mu.RUnlock()
		return ready
	}
	c.mu.RUnlock()

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.base+"/-/ready", nil)
	if err != nil {
		return false
	}
	c.authorize(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.setReady(false)
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 256))
	c.setReady(resp.StatusCode == http.StatusOK)
	return c.ready
}

// Query 瞬时查询，返回命中的一组样本
func (c *PrometheusClient) Query(ctx context.Context, expr string) ([]Sample, error) {
	data, err := c.doQuery(ctx, "/api/v1/query", url.Values{"query": {expr}})
	if err != nil {
		return nil, err
	}

	samples := make([]Sample, 0, len(data.Result))
	for _, raw := range data.Result {
		var res vectorResult
		if err := json.Unmarshal(raw, &res); err != nil {
			continue
		}
		value, ts, err := parsePair(res.Value[:])
		if err != nil {
			continue
		}
		samples = append(samples, Sample{Labels: res.Metric, Value: value, Timestamp: ts})
	}
	return samples, nil
}

// QueryRange 区间查询，返回多条时间序列
func (c *PrometheusClient) QueryRange(ctx context.Context, expr string, start, end time.Time, step time.Duration) ([]Series, error) {
	if step <= 0 {
		step = time.Minute
	}
	params := url.Values{
		"query": {expr},
		"start": {strconv.FormatInt(start.Unix(), 10)},
		"end":   {strconv.FormatInt(end.Unix(), 10)},
		"step":  {strconv.FormatInt(int64(step/time.Second), 10)},
	}
	data, err := c.doQuery(ctx, "/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}

	series := make([]Series, 0, len(data.Result))
	for _, raw := range data.Result {
		var res matrixResult
		if err := json.Unmarshal(raw, &res); err != nil {
			continue
		}
		item := Series{Labels: res.Metric, Points: make([]Point, 0, len(res.Values))}
		for _, pair := range res.Values {
			value, ts, err := parsePair(pair[:])
			if err != nil {
				continue
			}
			item.Points = append(item.Points, Point{Timestamp: ts, Value: value})
		}
		series = append(series, item)
	}
	return series, nil
}

func (c *PrometheusClient) doQuery(ctx context.Context, path string, params url.Values) (*apiData, error) {
	reqURL := c.base + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	resp, err := c.client.Do(req)
	if err != nil {
		c.setReady(false)
		return nil, fmt.Errorf("%w: %v", ErrDatasourceUnavailable, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		c.setReady(false)
		var apiErr apiResponse
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Error != "" {
			return nil, fmt.Errorf("查询监控数据失败: %s", apiErr.Error)
		}
		return nil, fmt.Errorf("%w: 查询返回状态码 %d", ErrDatasourceUnavailable, resp.StatusCode)
	}
	c.setReady(true)

	var parsed apiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("解析监控数据失败: %w", err)
	}
	if parsed.Status != "success" {
		return nil, fmt.Errorf("查询监控数据失败: %s", parsed.Error)
	}

	var data apiData
	if err := json.Unmarshal(parsed.Data, &data); err != nil {
		return nil, fmt.Errorf("解析监控数据失败: %w", err)
	}
	return &data, nil
}

// parsePair 解析 [时间戳(秒, 可含小数), 字符串值] 形式的返回对
func parsePair(pair []json.RawMessage) (float64, int64, error) {
	if len(pair) != 2 {
		return 0, 0, fmt.Errorf("非法的数据点")
	}
	var tsFloat float64
	if err := json.Unmarshal(pair[0], &tsFloat); err != nil {
		return 0, 0, err
	}
	var raw string
	if err := json.Unmarshal(pair[1], &raw); err != nil {
		return 0, 0, err
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, 0, err
	}
	return value, int64(tsFloat), nil
}

func (c *PrometheusClient) authorize(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c *PrometheusClient) setReady(ready bool) {
	c.mu.Lock()
	c.ready = ready
	c.checked = time.Now()
	c.mu.Unlock()
}
