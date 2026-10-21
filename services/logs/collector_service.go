package logs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/instance"
	"github.com/zhany/ops-go/services/logs/datasource"
)

// execOnHost 复用任务引擎的 SSH 执行能力，抽成变量便于替换
var execOnHost = instance.ExecuteOnHost

// LogCollectorService 日志采集服务：SSH 增量拉取远端日志 → 写入 Loki
type LogCollectorService struct {
	Interval       time.Duration
	MaxConcurrency int
	client         *datasource.Client

	lastAttempt sync.Map
	running     atomic.Bool
	lokiOK      bool
	lokiOKMu    sync.Mutex
}

// NewLogCollectorService 创建日志采集服务
func NewLogCollectorService() *LogCollectorService {
	return &LogCollectorService{
		Interval:       time.Duration(datasource.CollectIntervalSec()) * time.Second,
		MaxConcurrency: datasource.CollectConcurrency(),
		client:         datasource.NewClient(),
	}
}

// CollectResult 手动拉取一次的即时结果
type CollectResult struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	DurationMs int64  `json:"durationMs"`
}

// Start 启动采集定时任务
func (s *LogCollectorService) Start() {
	log.Printf("日志采集服务已启动，轮询间隔 %v，并发上限 %d，Loki: %s",
		s.Interval, s.MaxConcurrency, s.client.BaseURL())

	time.Sleep(6 * time.Second)
	s.collectOnce()

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for range ticker.C {
		s.collectOnce()
	}
}

// collectOnce 执行一轮采集
func (s *LogCollectorService) collectOnce() {
	if s.running.Swap(true) {
		log.Println("日志采集：上一轮尚未结束，跳过本轮")
		return
	}
	defer s.running.Store(false)

	ctx := context.Background()
	var sources []models.OpsLogSource
	if err := models.DB.Where("del_flag = ? AND enabled = ?", "0", "1").Find(&sources).Error; err != nil {
		log.Println("日志采集：查询采集源失败:", err)
		return
	}
	// 未配置采集源时保持静默，不产生日志噪声
	if len(sources) == 0 {
		return
	}

	if !s.client.Available(ctx) {
		s.logLokiState(false, "Loki 不可达，本轮跳过采集")
		return
	}
	s.logLokiState(true, "Loki 已恢复，继续采集")

	due := pickDueSources(sources, time.Now(), s.snapshotAttempts(), int(s.Interval/time.Second))
	if len(due) == 0 {
		return
	}

	success, failed := s.runBatch(ctx, due)
	log.Printf("日志采集完成：采集源 %d 个，本轮拉取 %d 个，成功 %d 个，失败 %d 个",
		len(sources), len(due), success, failed)
}

func (s *LogCollectorService) snapshotAttempts() map[int]time.Time {
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

// pickDueSources 选出到期的采集源，失败源按同样间隔退避，避免持续打爆远端
func pickDueSources(sources []models.OpsLogSource, now time.Time, attempts map[int]time.Time, defaultInterval int) []models.OpsLogSource {
	intervalOf := func(source models.OpsLogSource) time.Duration {
		sec := source.IntervalSec
		if sec <= 0 {
			sec = defaultInterval
		}
		return time.Duration(sec) * time.Second
	}

	due := make([]models.OpsLogSource, 0, len(sources))
	for _, source := range sources {
		last := attempts[source.ID]
		if source.LastCollectAt != nil && (last.IsZero() || source.LastCollectAt.After(last)) {
			last = *source.LastCollectAt
		}
		if !last.IsZero() && now.Sub(last) < intervalOf(source) {
			continue
		}
		due = append(due, source)
	}
	return due
}

// runBatch 并发拉取到期采集源，返回成功与失败数量
func (s *LogCollectorService) runBatch(ctx context.Context, sources []models.OpsLogSource) (int, int) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.MaxConcurrency)
	resultChan := make(chan bool, len(sources))

	now := time.Now()
	for i := range sources {
		source := sources[i]
		s.lastAttempt.Store(source.ID, now)

		wg.Add(1)
		semaphore <- struct{}{}
		go func(item models.OpsLogSource) {
			defer wg.Done()
			defer func() { <-semaphore }()

			err := s.collectSource(ctx, &item)
			s.saveResult(&item, err)
			resultChan <- err == nil
		}(source)
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

// CollectNow 立即拉取单个采集源一次，用于界面「试采集」排障。
// 停用源同样允许手动拉取，位点照常推进。
func (s *LogCollectorService) CollectNow(ctx context.Context, sourceId int) (*CollectResult, error) {
	var source models.OpsLogSource
	if err := models.DB.Where("id = ? AND del_flag = ?", sourceId, "0").First(&source).Error; err != nil {
		return nil, errors.New("采集源不存在")
	}

	start := time.Now()
	err := s.collectSource(ctx, &source)
	durationMs := time.Since(start).Milliseconds()

	s.lastAttempt.Store(source.ID, start)
	s.saveResult(&source, err)

	if err != nil {
		return &CollectResult{Success: false, DurationMs: durationMs, Message: fmt.Sprintf("拉取失败: %v", err)}, nil
	}
	return &CollectResult{Success: true, DurationMs: durationMs,
		Message: fmt.Sprintf("拉取成功：主机 %s 文件 %s，耗时 %d 毫秒", source.InstanceIp, source.Path, durationMs)}, nil
}

// collectSource 拉取单个采集源的一次增量并写入 Loki
func (s *LogCollectorService) collectSource(ctx context.Context, source *models.OpsLogSource) error {
	task, err := buildTailTask(source)
	if err != nil {
		return err
	}

	output, execErr := execOnHost(task)
	if execErr != nil && errors.Is(execErr, context.DeadlineExceeded) {
		return fmt.Errorf("拉取超时: %v", execErr)
	}

	result, parseErr := datasource.ParseTailOutput(output)
	if execErr != nil {
		// 脚本失败时已打印 OPS_ERR 原因（文件不存在/不可读），比 "exit status 3" 更能定位问题
		if parseErr != nil {
			return parseErr
		}
		return fmt.Errorf("远程拉取失败: %v", execErr)
	}
	if parseErr != nil {
		return parseErr
	}

	previousInode, previousPos := source.LastInode, source.LastPos
	source.LastInode = result.Inode
	source.LastPos = result.Pos
	source.LastFile = result.FilePath
	source.LastLines = len(result.Lines)
	source.LastDrop = result.Dropped

	if len(result.Lines) == 0 {
		return nil
	}
	if err := s.client.Push(ctx, buildStreams(source, result.Lines)); err != nil {
		// 写入失败则退回原位移点，下一轮重新拉取同一段，保证日志不丢
		source.LastInode, source.LastPos = previousInode, previousPos
		return err
	}
	return nil
}

// buildStreams 按标签集合分组日志行：Loki 的一个 stream = job/host/source/path 标签 + 级别
func buildStreams(source *models.OpsLogSource, lines []string) []datasource.Stream {
	base := map[string]string{"job": source.Job, "host": source.InstanceIp}
	if value := datasource.SafeLabelValue(source.Source); value != "" {
		base["source"] = value
	}
	if value := datasource.SafeLabelValue(source.Path); value != "" {
		base["path"] = value
	}
	for key, value := range parseSourceLabels(source.Labels) {
		// 非法标签名会被 Loki 整批拒绝，保留标签会顶掉采集结果，配置写错时丢弃该标签而不是中断采集
		if !validLabelName(key) || reservedLabel[key] || datasource.SafeLabelValue(value) == "" {
			continue
		}
		base[key] = value
	}

	grouped := make(map[string]*datasource.Stream)
	order := make([]string, 0)
	now := time.Now()

	for _, line := range lines {
		labels := make(map[string]string, len(base)+1)
		for key, value := range base {
			labels[key] = value
		}
		if level := datasource.DetectLevel(line, source.LevelRegex); level != "" {
			labels["level"] = level
		}

		key := labelsKey(labels)
		stream, exists := grouped[key]
		if !exists {
			stream = &datasource.Stream{Labels: labels}
			grouped[key] = stream
			order = append(order, key)
		}

		timestamp := datasource.ParseLineTime(line)
		if timestamp.IsZero() {
			timestamp = now
		}
		stream.Entries = append(stream.Entries, datasource.LogEntry{Line: line, Timestamp: timestamp})
	}

	sort.Strings(order)
	streams := make([]datasource.Stream, 0, len(order))
	for _, key := range order {
		streams = append(streams, *grouped[key])
	}
	return streams
}

// labelsKey 标签集合的稳定指纹，用于把同标签集的日志行归到同一个流
func labelsKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(labels[key])
		builder.WriteByte(0)
	}
	return builder.String()
}

func parseSourceLabels(raw string) map[string]string {
	labels := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return labels
	}
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return map[string]string{}
	}
	return labels
}

// reservedLabel 采集链路自身写入的标签，不允许被自定义标签覆盖
var reservedLabel = map[string]bool{"job": true, "host": true, "source": true, "path": true, "level": true}

// buildTailTask 解析主机与凭证，把读取位点作为参数注入拉取脚本
func buildTailTask(source *models.OpsLogSource) (instance.ExecTask, error) {
	var host models.OpsInstance
	if err := models.DB.Where("id = ? AND del_flag = ? AND status = ?", source.InstanceId, "0", "1").
		First(&host).Error; err != nil {
		return instance.ExecTask{}, errors.New("主机不存在或已禁用")
	}

	keyId := source.KeyId
	if keyId == 0 {
		var auth models.OpsUserInstanceKeyAuth
		if err := models.DB.Where("instance_id = ? AND del_flag = ?", source.InstanceId, "0").
			First(&auth).Error; err == nil {
			keyId = auth.KeyId
		}
	}
	if keyId == 0 {
		return instance.ExecTask{}, errors.New("主机未绑定采集凭证")
	}

	var key models.OpsKey
	if err := models.DB.Where("id = ? AND del_flag = ? AND status = ?", keyId, "0", "1").
		First(&key).Error; err != nil {
		return instance.ExecTask{}, errors.New("采集凭证不存在或已禁用")
	}

	timeout := source.TimeoutSec
	if timeout <= 0 {
		timeout = datasource.CollectTimeoutSec()
	}
	maxBytes := source.MaxBytes
	if maxBytes <= 0 {
		maxBytes = datasource.LogMaxBytes()
	}
	backBytes := source.BackBytes
	if backBytes < 0 {
		backBytes = 0
	} else if backBytes == 0 {
		backBytes = datasource.InitialBackBytes()
	}

	// 位点参数为整数字面量，路径来自配置表，必须做 shell 转义后再拼接
	args := fmt.Sprintf("set -- %s %s %s %d %d\n",
		shellQuote(source.Path),
		strconv.FormatInt(source.LastPos, 10),
		strconv.FormatInt(source.LastInode, 10),
		maxBytes, backBytes)

	return instance.ExecTask{
		HostId:      uint64(source.ID),
		InstanceIP:  host.Ip,
		Port:        key.Port,
		User:        key.User,
		Credentials: key.Credentials, // 保持密文，由执行引擎解密
		KeyType:     key.Type,
		CommandType: models.TaskTypeScript,
		Content:     args + datasource.TailScript(),
		ScriptLang:  "shell",
		Timeout:     timeout,
	}, nil
}

// shellQuote 单引号包裹并转义内部单引号，阻断路径中的命令注入
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// saveResult 回写采集状态与读取位点，错误信息按字段长度截断
func (s *LogCollectorService) saveResult(source *models.OpsLogSource, err error) {
	updates := map[string]any{
		"last_inode": source.LastInode,
		"last_pos":   source.LastPos,
		"last_file":  truncate(source.LastFile, 512),
		"last_lines": source.LastLines,
		"last_drop":  source.LastDrop,
		"last_error": "",
	}
	if err != nil {
		updates["last_status"] = models.CollectStatusFail
		updates["last_error"] = truncate(err.Error(), 500)
	} else {
		now := time.Now()
		updates["last_status"] = models.CollectStatusOk
		updates["last_collect_at"] = &now
	}
	if dbErr := models.DB.Model(&models.OpsLogSource{}).Where("id = ?", source.ID).Updates(updates).Error; dbErr != nil {
		log.Printf("日志采集：回写状态失败 sourceId=%d: %v", source.ID, dbErr)
	}
}

// logLokiState 仅在可达状态翻转时输出日志，避免 Loki 故障期间按轮刷屏
func (s *LogCollectorService) logLokiState(ok bool, msg string) {
	s.lokiOKMu.Lock()
	defer s.lokiOKMu.Unlock()
	if s.lokiOK == ok {
		return
	}
	s.lokiOK = ok
	if msg != "" {
		log.Println("日志采集:", msg)
	}
}
