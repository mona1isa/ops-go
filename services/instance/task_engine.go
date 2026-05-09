package instance

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/utils"
	"golang.org/x/crypto/ssh"
)

const maxConcurrentHosts = 50

// ExecTask 执行任务定义
type ExecTask struct {
	HostId      uint64
	InstanceIP  string
	Port        int
	User        string
	Credentials string // 加密的凭证
	KeyType     int    // 1=密码 2=密钥
	CommandType int8   // 1=命令 2=脚本 3=文件
	Content     string // 命令内容或脚本内容
	ScriptLang  string // shell/python
	SrcPath     string // 源文件路径(类型3)
	DestPath    string // 目标路径(类型3)
	Timeout     int    // 秒
}

// RunExecution 异步执行任务（快速执行 / 模板执行）
func RunExecution(executionId uint64) {
	service := TaskExecutionService{}

	execution, err := service.GetByID(executionId)
	if err != nil {
		log.Printf("[TaskEngine] 加载执行记录失败 executionId=%d: %v", executionId, err)
		return
	}

	// 加载执行内容
	taskType, content, scriptLang, srcPath, destPath := loadExecContent(execution)

	// 更新执行状态为运行中
	if err := service.StartExecution(executionId); err != nil {
		log.Printf("[TaskEngine] 启动执行失败 executionId=%d: %v", executionId, err)
		return
	}

	// 加载主机列表
	hosts, err := service.GetExecutionHosts(executionId)
	if err != nil {
		log.Printf("[TaskEngine] 加载主机列表失败 executionId=%d: %v", executionId, err)
		return
	}

	// 并发执行
	runHostsConcurrent(executionId, 0, hosts, taskType, content, scriptLang, srcPath, destPath, execution.Timeout)

	// 汇总结果
	if err := service.FinishExecution(executionId); err != nil {
		log.Printf("[TaskEngine] 汇总结果失败 executionId=%d: %v", executionId, err)
	}
}

// RunPipelineExecution 异步执行编排任务
func RunPipelineExecution(executionId uint64) {
	service := TaskExecutionService{}

	execution, err := service.GetByID(executionId)
	if err != nil {
		log.Printf("[TaskEngine] 加载执行记录失败 executionId=%d: %v", executionId, err)
		return
	}

	// 加载编排步骤
	var steps []models.OpsPipelineStep
	models.DB.Where("pipeline_id = ?", execution.SourceId).Order("step_order asc").Find(&steps)
	if len(steps) == 0 {
		log.Printf("[TaskEngine] 编排无步骤 pipelineId=%d", execution.SourceId)
		now := time.Now()
		models.DB.Model(&models.OpsTaskExecution{}).Where("id = ?", executionId).Updates(map[string]interface{}{
			"status":      models.ExecStatusAllFail,
			"finished_at": &now,
		})
		return
	}

	// 更新执行状态为运行中
	if err := service.StartExecution(executionId); err != nil {
		log.Printf("[TaskEngine] 启动执行失败 executionId=%d: %v", executionId, err)
		return
	}

	// 加载模板主机列表（编排执行时创建的主机记录作为模板）
	templateHosts, err := service.GetExecutionHosts(executionId)
	if err != nil {
		log.Printf("[TaskEngine] 加载主机列表失败 executionId=%d: %v", executionId, err)
		return
	}

	// 按步骤顺序执行
	allAborted := false
	for _, step := range steps {
		if allAborted {
			// 创建跳过的步骤执行记录
			stepExec := createStepExecution(executionId, step)
			markStepHostsSkipped(executionId, stepExec.ID)
			now := time.Now()
			models.DB.Model(&models.OpsStepExecution{}).Where("id = ?", stepExec.ID).Updates(map[string]interface{}{
				"status":      models.StepStatusSkipped,
				"finished_at": &now,
			})
			continue
		}

		// 加载步骤模板内容
		tpl := loadTemplate(step.TemplateId)
		if tpl == nil {
			log.Printf("[TaskEngine] 步骤模板不存在 stepId=%d templateId=%d", step.ID, step.TemplateId)
			// 创建失败的步骤执行记录
			stepExec := createStepExecution(executionId, step)
			now := time.Now()
			models.DB.Model(&models.OpsStepExecution{}).Where("id = ?", stepExec.ID).Updates(map[string]interface{}{
				"status":      models.StepStatusFail,
				"finished_at": &now,
			})
			if step.OnFailure == models.OnFailureAbort {
				allAborted = true
			}
			continue
		}

		// 创建步骤执行记录
		stepExec := createStepExecution(executionId, step)

		// 为该步骤克隆主机执行记录
		stepHosts, err := service.CloneStepExecutionHosts(executionId, stepExec.ID, templateHosts)
		if err != nil {
			log.Printf("[TaskEngine] 克隆主机记录失败 stepExecId=%d: %v", stepExec.ID, err)
			now := time.Now()
			models.DB.Model(&models.OpsStepExecution{}).Where("id = ?", stepExec.ID).Updates(map[string]interface{}{
				"status":      models.StepStatusFail,
				"finished_at": &now,
			})
			if step.OnFailure == models.OnFailureAbort {
				allAborted = true
			}
			continue
		}

		// 执行该步骤
		runHostsConcurrent(executionId, stepExec.ID, stepHosts, tpl.Type, tpl.Content, tpl.ScriptLang, tpl.SrcPath, tpl.DestPath, execution.Timeout)

		// 检查步骤执行结果
		abort := checkStepResult(executionId, stepExec.ID)
		now := time.Now()
		stepStatus := models.StepStatusSuccess
		if abort {
			stepStatus = models.StepStatusFail
		}
		models.DB.Model(&models.OpsStepExecution{}).Where("id = ?", stepExec.ID).Updates(map[string]interface{}{
			"status":      stepStatus,
			"finished_at": &now,
		})

		if abort && step.OnFailure == models.OnFailureAbort {
			allAborted = true
		}
	}

	// 汇总结果
	if err := service.FinishExecution(executionId); err != nil {
		log.Printf("[TaskEngine] 汇总结果失败 executionId=%d: %v", executionId, err)
	}
}

// loadExecContent 加载执行内容
func loadExecContent(execution *models.OpsTaskExecution) (taskType int8, content string, scriptLang string, srcPath string, destPath string) {
	switch execution.Type {
	case models.ExecTypeQuickCommand, models.ExecTypeQuickScript, models.ExecTypeQuickFile:
		// 快速执行，内容保存在执行记录中
		return execution.Type, execution.Content, execution.ScriptLang, execution.SrcPath, execution.DestPath
	case models.ExecTypeTemplate:
		// 模板执行，从模板加载
		tpl := loadTemplate(execution.SourceId)
		if tpl != nil {
			return tpl.Type, tpl.Content, tpl.ScriptLang, tpl.SrcPath, tpl.DestPath
		}
	case models.ExecTypePipeline:
		// 编排执行由 RunPipelineExecution 处理
	}
	return execution.Type, "", "", "", ""
}

// loadTemplate 加载任务模板
func loadTemplate(templateId int) *models.OpsTaskTemplate {
	var tpl models.OpsTaskTemplate
	if err := models.DB.Where("id = ? AND del_flag = ?", templateId, 0).First(&tpl).Error; err != nil {
		return nil
	}
	return &tpl
}

// createStepExecution 创建步骤执行记录
func createStepExecution(executionId uint64, step models.OpsPipelineStep) *models.OpsStepExecution {
	now := time.Now()
	stepExec := &models.OpsStepExecution{
		ExecutionId: executionId,
		StepId:      step.ID,
		StepName:    step.StepName,
		TemplateId:  step.TemplateId,
		Status:      models.StepStatusRunning,
		StartedAt:   &now,
	}
	models.DB.Create(stepExec)
	return stepExec
}

// checkStepResult 检查步骤执行结果，返回是否应该中止
func checkStepResult(executionId uint64, stepExecId uint64) bool {
	var hosts []models.OpsExecutionHost
	models.DB.Where("execution_id = ? AND step_exec_id = ?", executionId, stepExecId).Find(&hosts)
	for _, h := range hosts {
		if h.Status == models.HostStatusFail || h.Status == models.HostStatusTimeout {
			return true
		}
	}
	return false
}

// markStepHostsSkipped 标记步骤相关的未完成主机为跳过
func markStepHostsSkipped(executionId uint64, stepExecId uint64) {
	models.DB.Model(&models.OpsExecutionHost{}).
		Where("execution_id = ? AND step_exec_id = ? AND status IN ?", executionId, stepExecId, []int8{models.HostStatusPending, models.HostStatusRunning}).
		Update("status", models.HostStatusSkipped)
}

// runHostsConcurrent 并发在多台主机上执行任务
func runHostsConcurrent(executionId uint64, stepExecId uint64, hosts []models.OpsExecutionHost, taskType int8, content string, scriptLang string, srcPath string, destPath string, timeout int) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentHosts)

	service := TaskExecutionService{}

	for i := range hosts {
		host := hosts[i]
		if host.Status != models.HostStatusPending {
			continue
		}

		// 加载凭证
		var key models.OpsKey
		if err := models.DB.Where("id = ? AND del_flag = ?", host.KeyId, 0).First(&key).Error; err != nil {
			now := time.Now()
			models.DB.Model(&models.OpsExecutionHost{}).Where("id = ?", host.ID).Updates(map[string]interface{}{
				"status":      models.HostStatusFail,
				"error_msg":   "凭证不存在或已删除",
				"finished_at": &now,
			})
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(h models.OpsExecutionHost) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			task := ExecTask{
				HostId:      h.ID,
				InstanceIP:  h.InstanceIP,
				Port:        key.Port,
				User:        key.User,
				Credentials: key.Credentials,
				KeyType:     key.Type,
				CommandType: taskType,
				Content:     content,
				ScriptLang:  scriptLang,
				SrcPath:     srcPath,
				DestPath:     destPath,
				Timeout:     timeout,
			}

			// 设置主机开始时间
			now := time.Now()
			models.DB.Model(&models.OpsExecutionHost{}).Where("id = ?", h.ID).Updates(map[string]interface{}{
				"status":     models.HostStatusRunning,
				"started_at": &now,
			})

			// 执行
			output, err := ExecuteOnHost(task)

			if err != nil {
				hostStatus := int8(models.HostStatusFail)
				if err == context.DeadlineExceeded {
					hostStatus = models.HostStatusTimeout
				}
				service.UpdateHostResult(h.ID, hostStatus, output, err.Error())
			} else {
				service.UpdateHostResult(h.ID, models.HostStatusSuccess, output, "")
			}
		}(host)
	}

	wg.Wait()
}

// ExecuteOnHost SSH到主机执行任务
func ExecuteOnHost(task ExecTask) (string, error) {
	timeout := task.Timeout
	if timeout <= 0 {
		timeout = 300
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 解密凭证
	credentials, err := utils.DecryptKey(task.Credentials)
	if err != nil {
		return "", fmt.Errorf("解密凭证失败: %w", err)
	}

	// 构建SSH认证
	var authMethods []ssh.AuthMethod
	if task.KeyType == 2 {
		signer, err := ssh.ParsePrivateKey([]byte(credentials))
		if err != nil {
			return "", fmt.Errorf("解析SSH密钥失败: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		authMethods = append(authMethods, ssh.Password(credentials))
	}

	// 连接SSH
	port := task.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(task.InstanceIP, strconv.Itoa(port))

	config := &ssh.ClientConfig{
		User:            task.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", fmt.Errorf("连接主机失败: %w", err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("SSH握手失败: %w", err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	switch task.CommandType {
	case models.TaskTypeCommand:
		return executeCommand(ctx, client, task.Content)
	case models.TaskTypeScript:
		return executeScript(ctx, client, task.Content, task.ScriptLang)
	case models.TaskTypeFile:
		return executeFileTransfer(ctx, client, task.SrcPath, task.DestPath)
	default:
		return "", fmt.Errorf("不支持的任务类型: %d", task.CommandType)
	}
}

// executeCommand 在远程主机执行命令
func executeCommand(ctx context.Context, client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	type result struct {
		output []byte
		err    error
	}
	done := make(chan result, 1)

	go func() {
		out, e := session.CombinedOutput(command)
		done <- result{output: out, err: e}
	}()

	select {
	case <-ctx.Done():
		session.Close()
		return "", ctx.Err()
	case r := <-done:
		output := string(r.output)
		if r.err != nil {
			return output, r.err
		}
		return output, nil
	}
}

// executeScript 在远程主机执行脚本
func executeScript(ctx context.Context, client *ssh.Client, scriptContent string, scriptLang string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 构建执行命令
	var command string
	tmpFile := fmt.Sprintf("/tmp/ops_exec_%d.sh", time.Now().UnixNano())

	switch scriptLang {
	case "python":
		tmpFile = strings.Replace(tmpFile, ".sh", ".py", 1)
		// 写入脚本内容并执行
		command = fmt.Sprintf("cat > %s << 'OPS_SCRIPT_EOF'\n%s\nOPS_SCRIPT_EOF\npython3 %s && rm -f %s", tmpFile, scriptContent, tmpFile, tmpFile)
	default:
		// shell
		command = fmt.Sprintf("cat > %s << 'OPS_SCRIPT_EOF'\n%s\nOPS_SCRIPT_EOF\nchmod +x %s && /bin/bash %s && rm -f %s", tmpFile, scriptContent, tmpFile, tmpFile, tmpFile)
	}

	type result struct {
		output []byte
		err    error
	}
	done := make(chan result, 1)

	go func() {
		out, e := session.CombinedOutput(command)
		done <- result{output: out, err: e}
	}()

	select {
	case <-ctx.Done():
		session.Close()
		return "", ctx.Err()
	case r := <-done:
		output := string(r.output)
		if r.err != nil {
			return output, r.err
		}
		return output, nil
	}
}

// executeFileTransfer 通过SFTP上传文件到远程主机
func executeFileTransfer(ctx context.Context, client *ssh.Client, srcPath string, destPath string) (string, error) {
	// 打开本地源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return "", fmt.Errorf("获取源文件信息失败: %w", err)
	}

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return "", fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	// 确保远程目录存在
	remoteDir := filepath.Dir(destPath)
	sftpClient.MkdirAll(remoteDir)

	// 创建远程文件
	dstFile, err := sftpClient.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制文件内容
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("传输文件失败: %w", err)
	}

	// 设置文件权限
	dstFile.Chmod(srcInfo.Mode())

	return fmt.Sprintf("文件传输完成: %s -> %s (%d bytes)", srcPath, destPath, written), nil
}
