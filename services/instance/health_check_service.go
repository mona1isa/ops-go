package instance

import (
	"fmt"
	"github.com/zhany/ops-go/models"
	"log"
	"net"
	"sync"
	"time"
)

// HealthCheckService 主机健康检查服务
type HealthCheckService struct {
	// 探测间隔（默认 30 秒）
	Interval time.Duration
	// TCP 连接超时（默认 3 秒）
	Timeout time.Duration
	// 最大并发探测数
	MaxConcurrency int
}

// NewHealthCheckService 创建健康检查服务
func NewHealthCheckService() *HealthCheckService {
	return &HealthCheckService{
		Interval:       30 * time.Second,
		Timeout:        3 * time.Second,
		MaxConcurrency: 50,
	}
}

// Start 启动健康检查定时任务
func (s *HealthCheckService) Start() {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	// 首次立即执行一次
	s.checkAllInstances()

	for range ticker.C {
		s.checkAllInstances()
	}
}

// checkAllInstances 检查所有启用状态的主机
func (s *HealthCheckService) checkAllInstances() {
	var instances []models.OpsInstance
	if err := models.DB.Where("status = ? AND del_flag = ?", "1", 0).Find(&instances).Error; err != nil {
		log.Println("健康检查：查询主机列表失败:", err)
		return
	}

	if len(instances) == 0 {
		return
	}

	// 获取所有主机的绑定凭证以确定端口
	type instancePort struct {
		id   int
		ip   string
		port int
	}

	var instancePorts []instancePort
	for _, inst := range instances {
		port := s.getInstancePort(uint(inst.ID))
		instancePorts = append(instancePorts, instancePort{
			id:   int(inst.ID),
			ip:   inst.Ip,
			port: port,
		})
	}

	// 并发探测，使用信号量限制并发数
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.MaxConcurrency)

	type result struct {
		id     int
		online bool
	}
	resultChan := make(chan result, len(instancePorts))

	for _, inst := range instancePorts {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(inst instancePort) {
			defer wg.Done()
			defer func() { <-semaphore }()

			online := s.tcpDial(inst.ip, inst.port)
			resultChan <- result{id: inst.id, online: online}
		}(inst)
	}

	// 等待所有探测完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	var onlineIds []int
	var offlineIds []int
	for r := range resultChan {
		if r.online {
			onlineIds = append(onlineIds, r.id)
		} else {
			offlineIds = append(offlineIds, r.id)
		}
	}

	// 批量更新数据库
	if len(onlineIds) > 0 {
		if err := models.DB.Model(&models.OpsInstance{}).Where("id IN ?", onlineIds).Update("online_status", "1").Error; err != nil {
			log.Println("健康检查：更新在线状态失败:", err)
		}
	}
	if len(offlineIds) > 0 {
		if err := models.DB.Model(&models.OpsInstance{}).Where("id IN ?", offlineIds).Update("online_status", "0").Error; err != nil {
			log.Println("健康检查：更新离线状态失败:", err)
		}
	}

	log.Printf("健康检查完成：共 %d 台主机，在线 %d 台，离线 %d 台", len(instancePorts), len(onlineIds), len(offlineIds))
}

// tcpDial 通过 TCP Dial 检测主机端口是否可达
func (s *HealthCheckService) tcpDial(ip string, port int) bool {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, s.Timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// getInstancePort 获取主机的探测端口（优先使用绑定凭证的端口，默认 22）
func (s *HealthCheckService) getInstancePort(instanceId uint) int {
	var instanceKey models.OpsInstanceKey
	if err := models.DB.Where("instance_id = ?", instanceId).First(&instanceKey).Error; err != nil {
		return 22
	}

	var key models.OpsKey
	if err := models.DB.Where("id = ?", instanceKey.KeyId).First(&key).Error; err != nil {
		return 22
	}

	if key.Port > 0 {
		return key.Port
	}
	return 22
}
