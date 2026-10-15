package datasource

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrDatasourceUnavailable 监控数据源未配置或不可达
var ErrDatasourceUnavailable = errors.New("监控数据源不可用")

// Pushgateway 采集结果的写入端
type Pushgateway struct {
	base    string
	token   string
	client  *http.Client
	mu      sync.RWMutex
	ok      bool
	checked time.Time
}

// NewPushgateway 按环境变量构造 Pushgateway 客户端
func NewPushgateway() *Pushgateway {
	return &Pushgateway{
		base:   PushgatewayURL(),
		token:  pushgatewayToken(),
		client: newHTTPClient(pushTimeout()),
	}
}

// groupPath 指标分组路径：job 固定为 ops_node，instance 为主机 IP
func (p *Pushgateway) groupPath(instanceIP string) string {
	return fmt.Sprintf("/metrics/job/%s/instance/%s", MetricJob, url.PathEscape(instanceIP))
}

// Push 覆盖写入某台主机的整组指标
func (p *Pushgateway) Push(ctx context.Context, instanceIP, exposition string) error {
	if instanceIP == "" {
		return errors.New("主机IP不能为空")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, p.base+p.groupPath(instanceIP), strings.NewReader(exposition))
	if err != nil {
		return err
	}
	p.authorize(req)

	resp, err := p.client.Do(req)
	if err != nil {
		p.setState(false)
		return fmt.Errorf("%w: %v", ErrDatasourceUnavailable, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))

	if resp.StatusCode != http.StatusOK {
		p.setState(false)
		return fmt.Errorf("推送指标失败 status=%d body=%.200s", resp.StatusCode, body)
	}
	p.setState(true)
	return nil
}

// Delete 删除某台主机的指标分组，避免下线主机留下陈旧序列
func (p *Pushgateway) Delete(ctx context.Context, instanceIP string) error {
	if instanceIP == "" {
		return errors.New("主机IP不能为空")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, p.base+p.groupPath(instanceIP), nil)
	if err != nil {
		return err
	}
	p.authorize(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatasourceUnavailable, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 512))

	// Pushgateway 对不存在的分组返回 404，视为已清理
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("删除指标分组失败 status=%d", resp.StatusCode)
	}
	return nil
}

// Available 探测 Pushgateway 是否可达，结果缓存 30 秒避免逐台主机重试
func (p *Pushgateway) Available(ctx context.Context) bool {
	p.mu.RLock()
	if time.Since(p.checked) < 30*time.Second {
		ok := p.ok
		p.mu.RUnlock()
		return ok
	}
	p.mu.RUnlock()

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, p.base+"/", nil)
	if err != nil {
		return false
	}
	p.authorize(req)

	resp, err := p.client.Do(req)
	if err != nil {
		p.setState(false)
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 256))
	p.setState(resp.StatusCode == http.StatusOK)
	return p.ok
}

func (p *Pushgateway) authorize(req *http.Request) {
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
}

// GroupInstances 读取 Pushgateway 中属于本作业的主机列表，用于清理陈旧分组
// Pushgateway 会把 URL 分组的 job/instance 标签补全到 /metrics 输出里
func (p *Pushgateway) GroupInstances(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+"/metrics", nil)
	if err != nil {
		return nil, err
	}
	p.authorize(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatasourceUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("读取指标分组失败 status=%d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 32<<20))
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
	seen := make(map[string]struct{})
	jobMarker := `job="` + MetricJob + `"`
	instancePrefix := `instance="`

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || !strings.Contains(line, jobMarker) {
			continue
		}
		idx := strings.Index(line, instancePrefix)
		if idx < 0 {
			continue
		}
		rest := line[idx+len(instancePrefix):]
		end := strings.Index(rest, `"`)
		if end <= 0 {
			continue
		}
		seen[rest[:end]] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	instances := make([]string, 0, len(seen))
	for ip := range seen {
		instances = append(instances, ip)
	}
	sort.Strings(instances)
	return instances, nil
}

func (p *Pushgateway) setState(ok bool) {
	p.mu.Lock()
	p.ok = ok
	p.checked = time.Now()
	p.mu.Unlock()
}
