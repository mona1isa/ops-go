package main

import (
	"github.com/zhany/ops-go/bastion"
	"github.com/zhany/ops-go/controllers/instance"
	"github.com/zhany/ops-go/routers"
	cmdbService "github.com/zhany/ops-go/services/cmdb"
	instanceService "github.com/zhany/ops-go/services/instance"
	logService "github.com/zhany/ops-go/services/logs"
	monitorService "github.com/zhany/ops-go/services/monitor"
	systemService "github.com/zhany/ops-go/services/system"
	"log"
	"os"
)

func main() {
	// 注册 WebSocket 会话终止器
	instanceService.RegisterTerminator(instance.GetWebSocketTerminator())

	// 初始化高危指令规则（内置规则入库 + 加载到内存缓存），必须在堡垒机启动前完成，
	// 否则缓存为空，CheckCommand 会放行所有命令
	instanceService.InitDangerousCommands()
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
	// 启动证书/域名到期告警评估服务（扫描资产到期时间，复用监控告警通知链路）
	go cmdbService.NewCertAlertEvaluator().Start()
	// 补齐主机管理批量导入按钮权限
	instanceService.InitInstanceImportMenu()
	// 启动主机指标采集服务
	go monitorService.NewCollectorService().Start()
	// 启动告警规则评估服务
	go monitorService.NewEvaluatorService().Start()
	// 初始化日志中心菜单
	logService.InitLogMenu()
	// 按 sys_role_menu 补齐各角色的 Casbin 策略，使种子新增的菜单/按钮对非超管角色立即可用
	systemService.SyncRolePolicies()
	// 启动日志采集服务（SSH 增量拉取 → Loki）
	go logService.NewLogCollectorService().Start()
	// 启动日志告警评估服务
	go logService.NewLogAlertEvaluator().Start()
	// 启动登录日志保留期清理服务（LOGIN_LOG_RETENTION_DAYS，默认 180 天，0 为永久保留）
	go logService.NewLoginLogCleanupService().Start()
	// 启动Web服务
	r := routers.Init()
	err := r.Run(":" + os.Getenv("APP_PORT"))
	if err != nil {
		log.Println("服务启动异常：", err)
		return
	}
	log.Println("服务已成功启动.")

}
