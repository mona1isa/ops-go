package monitor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/instance"
	"github.com/zhany/ops-go/services/monitor/datasource"
)

// execOnHost 复用任务引擎的 SSH 执行能力，抽成变量便于替换
var execOnHost = instance.ExecuteOnHost

// 陈旧分组清理周期（轮）
const reconcileEveryCycles = uint64(10)

// CollectorService 主机指标采集服务
type CollectorService struct {
	// 采集轮询间隔
	Interval time.Duration
	// 单轮采集的最大并发主机数
	MaxConcurrency int
	// 指标写入端
	gw *datasource.Pushgateway

	// 每台主机最近一次采集尝试时间（含失败），用于失败退避
	lastAttempt sync.Map
	// 防止上一轮未结束时叠轮
	running atomic.Bool
	// 推送网关可达状态，仅在状态翻转时打印日志
	pushOK   bool
	pushOKMu sync.Mutex
	// 陈旧分组清理计数，避免每轮都拉取全量分组
	cycle uint64
}

// NewCollectorService 创建采集服务
func NewCollectorService() *CollectorService {
	return &CollectorService{
		Interval:       time.Duration(collectIntervalSec()) * time.Second,
		MaxConcurrency: collectConcurrency(),
		gw:             datasource.NewPushgateway(),
	}
}

// Start 启动采集定时任务
func (s *CollectorService) Start() {
	log.Printf("监控采集服务已启动，轮询间隔 %v，并发上限 %d，Pushgateway: %s",
		s.Interval, s.MaxConcurrency, datasource.PushgatewayURL())

	// 首轮延迟数秒，避开数据库迁移与健康检查的启动高峰
	time.Sleep(5 * time.Second)
	s.collectOnce()

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for range ticker.C {
		s.collectOnce()
	}
}

// collectOnce 执行一轮采集
func (s *CollectorService) collectOnce() {
	if s.running.Swap(true) {
		log.Println("监控采集：上一轮尚未结束，跳过本轮")
		return
	}
	defer s.running.Store(false)

	ctx := context.Background()
	var targets []models.OpsMonitorTarget
	if err := models.DB.Where("del_flag = ? AND enabled = ?", 0, "1").Find(&targets).Error; err != nil {
		log.Println("监控采集：查询采集目标失败:", err)
		return
	}
	// 未配置采集目标时保持静默，不产生日志噪声
	if len(targets) == 0 {
		return
	}

	if !s.gw.Available(ctx) {
		s.logPushState(false, "Pushgateway 不可达，本轮跳过采集")
		return
	}
	s.logPushState(true, "Pushgateway 已恢复，继续采集")

	due := pickDueTargets(targets, time.Now(), s.snapshotAttempts(), int(s.Interval/time.Second))
	if len(due) == 0 {
		return
	}

	success, failed := s.runBatch(ctx, due)

	// 周期性清理已停用/已删除主机残留的指标分组
	s.cycle++
	if s.cycle%reconcileEveryCycles == 0 {
		s.reconcileStale(ctx, targets)
	}

	log.Printf("监控采集完成：目标 %d 台，本轮采集 %d 台，成功 %d 台，失败 %d 台",
		len(targets), len(due), success, failed)
}

// snapshotAttempts 导出内存中的采集尝试时间
func (s *CollectorService) snapshotAttempts() map[int]time.Time {
	attempts := make(map[int]time.Time)
	s.lastAttempt.Range(func(key, value any) bool {
		if id, okKey := key.(int); okKey {
			if ts, okVal := value.(time.Time); okVal {
				attempts[id] = ts
			}
		}
		return true
	})
	return attempts
}

// pickDueTargets 选出到期需要采集的目标
// 采集间隔取目标自身配置；失败主机不会更新成功时间，靠尝试时间做同样间隔的退避
func pickDueTargets(targets []models.OpsMonitorTarget, now time.Time, attempts map[int]time.Time, defaultInterval int) []models.OpsMonitorTarget {
	intervalOf := func(target models.OpsMonitorTarget) time.Duration {
		sec := target.IntervalSec
		if sec <= 0 {
			sec = defaultInterval
		}
		return time.Duration(sec) * time.Second
	}

	due := make([]models.OpsMonitorTarget, 0, len(targets))
	for _, target := range targets {
		last := attempts[target.InstanceId]
		if target.LastCollectAt != nil && (last.IsZero() || target.LastCollectAt.After(last)) {
			last = *target.LastCollectAt
		}
		if !last.IsZero() && now.Sub(last) < intervalOf(target) {
			continue
		}
		due = append(due, target)
	}
	return due
}

// runBatch 并发采集到期目标，返回成功与失败数量
func (s *CollectorService) runBatch(ctx context.Context, targets []models.OpsMonitorTarget) (int, int) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.MaxConcurrency)
	resultChan := make(chan bool, len(targets))

	now := time.Now()
	for i := range targets {
		target := targets[i]
		s.lastAttempt.Store(target.InstanceId, now)

		wg.Add(1)
		semaphore <- struct{}{}
		go func(t models.OpsMonitorTarget) {
			defer wg.Done()
			defer func() { <-semaphore }()

			err := s.collectTarget(ctx, &t)
			s.saveResult(&t, err)
			resultChan <- err == nil
		}(target)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	success, failed := 0, 0
	for ok := range resultChan {
		if ok {
			success++
		} else {
			failed++
		}
	}
	return success, failed
}

// CollectResult 手动采集一次的即时结果
type CollectResult struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	DurationMs int64  `json:"durationMs"`
}

// CollectTargetNow 对单个采集目标立即执行一次采集，并回写采集状态。
// 独立于定时轮执行：Push 按主机分组幂等覆盖，与后台轮次并发无副作用；
// 停用目标也允许手动采集，便于排查连接与配置问题。
func (s *CollectorService) CollectTargetNow(ctx context.Context, targetId int) (*CollectResult, error) {
	var target models.OpsMonitorTarget
	if err := models.DB.Where("id = ? AND del_flag = ?", targetId, 0).First(&target).Error; err != nil {
		return nil, errors.New("采集目标不存在")
	}

	start := time.Now()
	err := s.collectTarget(ctx, &target)
	durationMs := time.Since(start).Milliseconds()

	s.lastAttempt.Store(target.InstanceId, start)
	s.saveResult(&target, err)

	if err != nil {
		return &CollectResult{Success: false, DurationMs: durationMs,
			Message: fmt.Sprintf("采集失败: %v", err)}, nil
	}
	return &CollectResult{Success: true, DurationMs: durationMs,
		Message: fmt.Sprintf("采集成功：主机 %s 连接正常，指标已写入数据源，耗时 %d 毫秒", target.InstanceIp, durationMs)}, nil
}

// collectTarget 采集单台主机并写入 Pushgateway
func (s *CollectorService) collectTarget(ctx context.Context, target *models.OpsMonitorTarget) error {
	task, err := buildExecTask(target)
	if err != nil {
		return err
	}

	output, err := execOnHost(task)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("采集超时: %v", err)
		}
		return fmt.Errorf("远程采集失败: %v", err)
	}

	payload, err := datasource.ParseCollectOutput(output)
	if err != nil {
		return err
	}
	return s.gw.Push(ctx, target.InstanceIp, datasource.BuildExposition(payload))
}

// buildExecTask 解析主机与凭证，构造远程执行任务
func buildExecTask(target *models.OpsMonitorTarget) (instance.ExecTask, error) {
	var host models.OpsInstance
	if err := models.DB.Where("id = ? AND del_flag = ? AND status = ?", target.InstanceId, 0, "1").
		First(&host).Error; err != nil {
		return instance.ExecTask{}, errors.New("主机不存在或已禁用")
	}

	keyId := target.KeyId
	if keyId == 0 {
		// 未显式指定凭证时，沿用主机授权关系中的第一个凭证
		var auth models.OpsUserInstanceKeyAuth
		if err := models.DB.Where("instance_id = ? AND del_flag = ?", target.InstanceId, 0).
			First(&auth).Error; err == nil {
			keyId = auth.KeyId
		}
	}
	if keyId == 0 {
		return instance.ExecTask{}, errors.New("主机未绑定采集凭证")
	}

	var key models.OpsKey
	if err := models.DB.Where("id = ? AND del_flag = ? AND status = ?", keyId, 0, "1").
		First(&key).Error; err != nil {
		return instance.ExecTask{}, errors.New("采集凭证不存在或已禁用")
	}

	timeout := target.TimeoutSec
	if timeout <= 0 {
		timeout = collectTimeoutSec()
	}

	return instance.ExecTask{
		HostId:      uint64(target.ID),
		InstanceIP:  host.Ip,
		Port:        key.Port,
		User:        key.User,
		Credentials: key.Credentials, // 保持密文，由执行引擎解密
		KeyType:     key.Type,
		CommandType: models.TaskTypeScript,
		Content:     datasource.CollectionScript(),
		ScriptLang:  "shell",
		Timeout:     timeout,
	}, nil
}

// saveResult 回写采集状态，错误信息按字段长度截断
func (s *CollectorService) saveResult(target *models.OpsMonitorTarget, err error) {
	updates := map[string]any{"last_error": ""}
	if err != nil {
		updates["last_status"] = models.CollectStatusFail
		updates["last_error"] = truncate(err.Error(), 500)
	} else {
		now := time.Now()
		updates["last_status"] = models.CollectStatusOk
		updates["last_collect_at"] = &now
	}
	if dbErr := models.DB.Model(&models.OpsMonitorTarget{}).Where("id = ?", target.ID).Updates(updates).Error; dbErr != nil {
		log.Printf("监控采集：回写采集状态失败 targetId=%d: %v", target.ID, dbErr)
	}
}

// reconcileStale 删除 Pushgateway 中已不再监控的主机分组
func (s *CollectorService) reconcileStale(ctx context.Context, active []models.OpsMonitorTarget) {
	pushed, err := s.gw.GroupInstances(ctx)
	if err != nil {
		log.Println("监控采集：读取指标分组失败:", err)
		return
	}

	keep := make(map[string]struct{}, len(active))
	for _, target := range active {
		keep[target.InstanceIp] = struct{}{}
	}

	for _, ip := range pushed {
		if _, ok := keep[ip]; ok {
			continue
		}
		if err := s.gw.Delete(ctx, ip); err != nil {
			log.Printf("监控采集：清理陈旧分组 %s 失败: %v", ip, err)
			continue
		}
		log.Printf("监控采集：已清理主机 %s 的陈旧指标分组", ip)
	}
}

// logPushState 仅在可达状态翻转时输出日志，避免数据源故障期间按轮刷屏
func (s *CollectorService) logPushState(ok bool, msg string) {
	s.pushOKMu.Lock()
	defer s.pushOKMu.Unlock()
	if s.pushOK == ok {
		return
	}
	s.pushOK = ok
	if msg != "" {
		log.Println("监控采集:", msg)
	}
}
