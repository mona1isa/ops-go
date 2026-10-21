package monitor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/monitor/datasource"
)

// MetricService 监控指标查询服务，只读代理 Prometheus HTTP API
type MetricService struct {
	client *datasource.PrometheusClient
}

// NewMetricService 构造指标查询服务
func NewMetricService() *MetricService {
	return &MetricService{client: datasource.NewPrometheusClient()}
}

// queryContext 单次查询的超时上下文
func queryContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(queryTimeoutSec())*time.Second)
}

// Query 执行瞬时查询，把数据源错误翻译成用户可读的中文提示
func (s *MetricService) Query(ctx context.Context, expr string) ([]datasource.Sample, error) {
	samples, err := s.client.Query(ctx, expr)
	if err != nil {
		return nil, readableQueryError(err)
	}
	return samples, nil
}

func readableQueryError(err error) error {
	if errors.Is(err, datasource.ErrDatasourceUnavailable) {
		return fmt.Errorf("监控数据源不可用，请确认 Prometheus(%s) 已启动", datasource.PrometheusURL())
	}
	return err
}

// DatasourceStatus 数据源与监控配置就绪情况
type DatasourceStatus struct {
	PrometheusURL   string   `json:"prometheusUrl"`
	PushgatewayURL  string   `json:"pushgatewayUrl"`
	PrometheusReady bool     `json:"prometheusReady"`
	PushgatewayOK   bool     `json:"pushgatewayReady"`
	TargetCount     int64    `json:"targetCount"`
	EnabledTargets  int64    `json:"enabledTargetCount"`
	RuleCount       int64    `json:"ruleCount"`
	FiringCount     int64    `json:"firingCount"`
	MetricNames     []string `json:"metricNames"`
	Message         string   `json:"message"`
}

// Status 汇总数据源可达性与监控配置规模
func (s *MetricService) Status(ctx context.Context) *DatasourceStatus {
	status := &DatasourceStatus{
		PrometheusURL:   datasource.PrometheusURL(),
		PushgatewayURL:  datasource.PushgatewayURL(),
		PrometheusReady: s.client.Available(ctx),
		PushgatewayOK:   datasource.NewPushgateway().Available(ctx),
		MetricNames:     datasource.PanelMetricNames(),
	}
	models.DB.Model(&models.OpsMonitorTarget{}).Where("del_flag = ?", "0").Count(&status.TargetCount)
	models.DB.Model(&models.OpsMonitorTarget{}).Where("del_flag = ? AND enabled = ?", "0", "1").Count(&status.EnabledTargets)
	models.DB.Model(&models.OpsMonitorRule{}).Where("del_flag = ? AND enabled = ?", "0", "1").Count(&status.RuleCount)
	models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("status = ?", models.AlertStatusFiring).Count(&status.FiringCount)

	switch {
	case !status.PrometheusReady:
		status.Message = fmt.Sprintf("Prometheus(%s) 不可用，请检查监控数据源地址", status.PrometheusURL)
	case !status.PushgatewayOK:
		status.Message = fmt.Sprintf("Pushgateway(%s) 不可用，主机指标无法写入", status.PushgatewayURL)
	case status.EnabledTargets == 0:
		status.Message = "尚未启用任何采集目标"
	default:
		status.Message = "监控数据源正常"
	}
	return status
}

// Hosts 已上报过指标的主机地址列表
func (s *MetricService) Hosts(ctx context.Context) ([]string, error) {
	samples, err := s.client.Query(ctx, datasource.MetricUptime)
	if err != nil {
		return nil, readableQueryError(err)
	}
	seen := map[string]bool{}
	hosts := make([]string, 0, len(samples))
	for _, sample := range samples {
		ip := sample.Instance()
		if ip == "" || seen[ip] {
			continue
		}
		seen[ip] = true
		hosts = append(hosts, ip)
	}
	sort.Strings(hosts)
	return hosts, nil
}

// MetricPoint 图表数据点
type MetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// MetricSeries 一条时间序列
type MetricSeries struct {
	Metric string            `json:"metric"`
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Points []MetricPoint     `json:"points"`
}

// HostSnapshot 主机当前指标快照
type HostSnapshot struct {
	Instance string         `json:"instance"`
	Metrics  []MetricSeries `json:"metrics"`
	Missing  bool           `json:"missing"`
	Message  string         `json:"message"`
}

// Snapshot 主机全指标瞬时快照
func (s *MetricService) Snapshot(ctx context.Context, instance string) (*HostSnapshot, error) {
	expr, err := datasource.InstantExpr(instance)
	if err != nil {
		return nil, err
	}
	samples, err := s.client.Query(ctx, expr)
	if err != nil {
		return nil, readableQueryError(err)
	}

	snapshot := &HostSnapshot{Instance: instance, Metrics: make([]MetricSeries, 0, len(samples))}
	for _, sample := range samples {
		name := sample.Labels["__name__"]
		if name == "" {
			continue
		}
		labels := withoutNoiseLabels(sample.Labels)
		snapshot.Metrics = append(snapshot.Metrics, MetricSeries{
			Metric: name,
			Name:   datasource.PanelMetricLabel(name),
			Labels: labels,
			Points: []MetricPoint{{Timestamp: sample.Timestamp, Value: sample.Value}},
		})
	}
	if len(snapshot.Metrics) == 0 {
		snapshot.Missing = true
		snapshot.Message = "该主机暂无监控数据，请确认采集目标已启用且采集成功"
	}
	return snapshot, nil
}

// RangeQuery 区间查询参数
type RangeQuery struct {
	Metric     string
	Instance   string
	Mountpoint string
	Device     string
	Start      time.Time
	End        time.Time
	StepSec    int
}

// Range 单主机单指标的趋势数据
func (s *MetricService) Range(ctx context.Context, query RangeQuery) ([]MetricSeries, error) {
	if !datasource.ValidPanelMetric(query.Metric) {
		return nil, errors.New("不支持的监控指标")
	}
	labels := map[string]string{}
	if query.Instance != "" {
		labels["instance"] = query.Instance
	}
	if query.Mountpoint != "" {
		labels["mountpoint"] = query.Mountpoint
	}
	if query.Device != "" {
		labels["device"] = query.Device
	}
	expr, err := datasource.Selector(query.Metric, labels)
	if err != nil {
		return nil, err
	}

	end := query.End
	if end.IsZero() {
		end = time.Now()
	}
	start := query.Start
	if start.IsZero() || !start.Before(end) {
		start = end.Add(-time.Hour)
	}
	step := rangeStep(query.StepSec, start, end)

	series, err := s.client.QueryRange(ctx, expr, start, end, step)
	if err != nil {
		return nil, readableQueryError(err)
	}
	result := make([]MetricSeries, 0, len(series))
	for _, item := range series {
		points := make([]MetricPoint, 0, len(item.Points))
		for _, point := range item.Points {
			points = append(points, MetricPoint{Timestamp: point.Timestamp, Value: point.Value})
		}
		result = append(result, MetricSeries{
			Metric: query.Metric,
			Name:   datasource.PanelMetricLabel(query.Metric),
			Labels: withoutNoiseLabels(item.Labels),
			Points: points,
		})
	}
	return result, nil
}

// rangeStep 控制返回点密度，避免长时间范围把大数组推给浏览器
func rangeStep(stepSec int, start, end time.Time) time.Duration {
	if stepSec > 0 {
		if stepSec < 5 {
			stepSec = 5
		}
		if stepSec > 3600 {
			stepSec = 3600
		}
		return time.Duration(stepSec) * time.Second
	}
	span := int(end.Sub(start).Seconds())
	if span <= 0 {
		return time.Minute
	}
	step := span / 300
	if step < 15 {
		step = 15
	}
	if step > 3600 {
		step = 3600
	}
	return time.Duration(step) * time.Second
}

// withoutNoiseLabels 去掉 Prometheus/Pushgateway 附加的采集元数据，只保留业务标签
func withoutNoiseLabels(labels map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range labels {
		switch key {
		case "job", "instance", "__name__":
			continue
		}
		result[key] = value
	}
	if name, ok := labels["__name__"]; ok {
		result["__name__"] = name
	}
	return result
}

// OverviewCount 统计当前告警规模，供首页与面板顶部使用
func OverviewCount() (map[string]int64, error) {
	counts := map[string]int64{}
	var firing, pending int64
	if err := models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("status = ?", models.AlertStatusFiring).Count(&firing).Error; err != nil {
		log.Println("统计告警中事件失败：", err)
		return nil, errors.New("统计告警事件失败")
	}
	if err := models.DB.Model(&models.OpsMonitorAlertEvent{}).Where("status = ?", models.AlertStatusPending).Count(&pending).Error; err != nil {
		log.Println("统计待确认告警失败：", err)
		return nil, errors.New("统计告警事件失败")
	}
	var collectFail int64
	if err := models.DB.Model(&models.OpsMonitorTarget{}).
		Where("del_flag = ? AND enabled = ? AND last_status = ?", "0", "1", models.CollectStatusFail).
		Count(&collectFail).Error; err != nil {
		log.Println("统计采集失败目标失败：", err)
	}
	var targets int64
	models.DB.Model(&models.OpsMonitorTarget{}).Where("del_flag = ? AND enabled = ?", "0", "1").Count(&targets)

	counts["firing"] = firing
	counts["pending"] = pending
	counts["collectFailed"] = collectFail
	counts["enabledTargets"] = targets
	return counts, nil
}
