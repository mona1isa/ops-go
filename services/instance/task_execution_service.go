package instance

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// TaskExecutionService 任务执行服务
type TaskExecutionService struct{}

// List 分页查询执行记录
func (s *TaskExecutionService) List(pageNum, pageSize int, status *int8, execType *int8, startAt, endAt string) (models.PageResult[models.OpsTaskExecution], error) {
	return models.Paginate[models.OpsTaskExecution](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		if status != nil {
			db = db.Where("status = ?", *status)
		}
		if execType != nil {
			db = db.Where("type = ?", *execType)
		}
		if startAt != "" {
			db = db.Where("created_at >= ?", startAt)
		}
		if endAt != "" {
			db = db.Where("created_at <= ?", endAt)
		}
		return db.Order("id desc")
	})
}

// GetByID 根据ID查询执行记录（含主机结果）
func (s *TaskExecutionService) GetByID(executionId uint64) (*models.OpsTaskExecution, error) {
	var execution models.OpsTaskExecution
	if err := models.DB.First(&execution, executionId).Error; err != nil {
		return nil, errors.New("执行记录不存在")
	}
	var hosts []models.OpsExecutionHost
	models.DB.Where("execution_id = ?", executionId).Order("id asc").Find(&hosts)
	return &execution, nil
}

// GetHostResult 查询单台主机执行结果
func (s *TaskExecutionService) GetHostResult(executionId uint64, instanceId int) (*models.OpsExecutionHost, error) {
	var host models.OpsExecutionHost
	if err := models.DB.Where("execution_id = ? AND instance_id = ?", executionId, instanceId).First(&host).Error; err != nil {
		return nil, errors.New("主机执行记录不存在")
	}
	return &host, nil
}

// CreateExecution 创建执行记录
func (s *TaskExecutionService) CreateExecution(name string, execType int8, sourceId int, userId int, userName string, instanceIds []int, keyId int, timeout int, content string, scriptLang string, srcPath string, destPath string) (*models.OpsTaskExecution, error) {
	if len(instanceIds) == 0 {
		return nil, errors.New("目标主机不能为空")
	}
	if timeout <= 0 {
		timeout = 300
	}

	now := time.Now()
	execution := &models.OpsTaskExecution{
		ExecutionNo: fmt.Sprintf("EXEC-%s-%04d", now.Format("20060102"), now.UnixNano()%10000),
		Name:        name,
		Type:        execType,
		SourceId:    sourceId,
		UserId:      userId,
		UserName:    userName,
		Status:      models.ExecStatusPending,
		TotalHosts:  len(instanceIds),
		Timeout:     timeout,
		Content:      content,
		ScriptLang:  scriptLang,
		SrcPath:     srcPath,
		DestPath:    destPath,
	}
	if err := models.DB.Create(execution).Error; err != nil {
		return nil, errors.New("创建执行记录失败")
	}

	// 解析凭证和主机信息，创建主机执行记录
	for _, instanceId := range instanceIds {
		host, err := s.buildExecutionHost(execution.ID, instanceId, keyId)
		if err != nil {
			log.Printf("创建主机执行记录失败 instanceId=%d: %v", instanceId, err)
			continue
		}
		models.DB.Create(host)
	}

	return execution, nil
}

// buildExecutionHost 构建主机执行记录
func (s *TaskExecutionService) buildExecutionHost(executionId uint64, instanceId int, keyId int) (*models.OpsExecutionHost, error) {
	var instance models.OpsInstance
	if err := models.DB.Where("id = ? AND del_flag = ? AND status = ?", instanceId, 0, "1").First(&instance).Error; err != nil {
		return nil, errors.New("主机不存在或不可用")
	}

	keyName := ""
	keyUser := ""
	resolvedKeyId := keyId

	if keyId == 0 {
		// 自动选择凭证
		var userAuths []models.OpsUserInstanceKeyAuth
		models.DB.Where("instance_id = ? AND del_flag = ?", instanceId, 0).Find(&userAuths)
		if len(userAuths) > 0 {
			resolvedKeyId = userAuths[0].KeyId
		}
	}

	if resolvedKeyId > 0 {
		var key models.OpsKey
		if err := models.DB.Where("id = ? AND del_flag = ?", resolvedKeyId, 0).First(&key).Error; err == nil {
			keyName = key.Name
			keyUser = key.User
		}
	}

	return &models.OpsExecutionHost{
		ExecutionId:  executionId,
		InstanceId:   instanceId,
		InstanceName: instance.Name,
		InstanceIP:   instance.Ip,
		KeyId:        resolvedKeyId,
		KeyName:      keyName,
		KeyUser:      keyUser,
		Status:       models.HostStatusPending,
	}, nil
}

// StartExecution 开始执行
func (s *TaskExecutionService) StartExecution(executionId uint64) error {
	now := time.Now()
	return models.DB.Model(&models.OpsTaskExecution{}).Where("id = ?", executionId).Updates(map[string]interface{}{
		"status":     models.ExecStatusRunning,
		"started_at": &now,
	}).Error
}

// UpdateHostResult 更新主机执行结果
func (s *TaskExecutionService) UpdateHostResult(hostId uint64, status int8, result string, errorMsg string) error {
	now := time.Now()
	var host models.OpsExecutionHost
	if err := models.DB.First(&host, hostId).Error; err != nil {
		return err
	}
	duration := 0
	if host.StartedAt != nil {
		duration = int(now.Sub(*host.StartedAt).Milliseconds())
	}
	return models.DB.Model(&host).Updates(map[string]interface{}{
		"status":      status,
		"result":      result,
		"error_msg":   errorMsg,
		"finished_at": &now,
		"duration":    duration,
	}).Error
}

// FinishExecution 完成执行（汇总各主机结果）
func (s *TaskExecutionService) FinishExecution(executionId uint64) error {
	var hosts []models.OpsExecutionHost
	models.DB.Where("execution_id = ?", executionId).Find(&hosts)

	successCount := 0
	failCount := 0
	for _, h := range hosts {
		if h.Status == models.HostStatusSuccess {
			successCount++
		} else if h.Status == models.HostStatusFail || h.Status == models.HostStatusTimeout {
			failCount++
		}
	}

	status := models.ExecStatusCompleted
	if failCount == len(hosts) {
		status = models.ExecStatusAllFail
	} else if failCount > 0 {
		status = models.ExecStatusPartialFail
	}

	now := time.Now()
	return models.DB.Model(&models.OpsTaskExecution{}).Where("id = ?", executionId).Updates(map[string]interface{}{
		"status":        status,
		"success_hosts": successCount,
		"fail_hosts":    failCount,
		"finished_at":   &now,
	}).Error
}

// CancelExecution 取消执行
func (s *TaskExecutionService) CancelExecution(executionId uint64, userId int) error {
	var execution models.OpsTaskExecution
	if err := models.DB.First(&execution, executionId).Error; err != nil {
		return errors.New("执行记录不存在")
	}
	if execution.Status != models.ExecStatusPending && execution.Status != models.ExecStatusRunning {
		return errors.New("当前状态不允许取消")
	}
	now := time.Now()
	models.DB.Model(&execution).Updates(map[string]interface{}{
		"status":      models.ExecStatusCancelled,
		"finished_at": &now,
	})
	// 将未完成的主机标记为跳过
	models.DB.Model(&models.OpsExecutionHost{}).
		Where("execution_id = ? AND status IN ?", executionId, []int8{models.HostStatusPending, models.HostStatusRunning}).
		Update("status", models.HostStatusSkipped)
	return nil
}

// GetExecutionHosts 获取执行的主机列表
func (s *TaskExecutionService) GetExecutionHosts(executionId uint64) ([]models.OpsExecutionHost, error) {
	var hosts []models.OpsExecutionHost
	if err := models.DB.Where("execution_id = ?", executionId).Order("id asc").Find(&hosts).Error; err != nil {
		return nil, err
	}
	return hosts, nil
}

// ParseUserId 解析用户ID字符串
func ParseUserId(userIdStr string) int {
	id, err := strconv.Atoi(userIdStr)
	if err != nil {
		return 0
	}
	return id
}
