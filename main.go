package main

import (
	"github.com/zhany/ops-go/bastion"
	"github.com/zhany/ops-go/controllers/instance"
	"github.com/zhany/ops-go/routers"
	cmdbService "github.com/zhany/ops-go/services/cmdb"
	instanceService "github.com/zhany/ops-go/services/instance"
	logService "github.com/zhany/ops-go/services/logs"
	monitorService "github.com/zhany/ops-go/services/monitor"
	"log"
	"os"
)

func main() {
	// 注册 WebSocket 会话终止器
	instanceService.RegisterTerminator(instance.GetWebSocketTerminator())

	// 启动堡垒机服务
	go bastion.Init()
	// 启动主机健康检查服务
	go instanceService.NewHealthCheckService().Start()
	// 启动审计录像自动清理服务
	go instanceService.NewRecordingCleanupService().Start()
	// 初始化监控告警菜单
	monitorService.InitMonitorMenu()
	// 初始化资产管理菜单
	cmdbService.InitCmdbMenu()
	// 补齐主机管理批量导入按钮权限
	instanceService.InitInstanceImportMenu()
	// 启动主机指标采集服务
	go monitorService.NewCollectorService().Start()
	// 启动告警规则评估服务
	go monitorService.NewEvaluatorService().Start()
	// 初始化日志中心菜单
	logService.InitLogMenu()
	// 启动日志采集服务（SSH 增量拉取 → Loki）
	go logService.NewLogCollectorService().Start()
	// 启动日志告警评估服务
	go logService.NewLogAlertEvaluator().Start()
	// 启动Web服务
	r := routers.Init()
	err := r.Run(":" + os.Getenv("APP_PORT"))
	if err != nil {
		log.Println("服务启动异常：", err)
		return
	}
	log.Println("服务已成功启动.")

}
