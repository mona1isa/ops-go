package main

import (
	"github.com/zhany/ops-go/bastion"
	"github.com/zhany/ops-go/controllers/instance"
	"github.com/zhany/ops-go/routers"
	instanceService "github.com/zhany/ops-go/services/instance"
	"log"
	"os"
)

func main() {
	// 注册 WebSocket 会话终止器
	instanceService.RegisterTerminator(instance.GetWebSocketTerminator())

	// 初始化高危指令规则
	instanceService.InitDangerousCommands()

	// 启动堡垒机服务
	go bastion.Init()
	// 启动Web服务
	r := routers.Init()
	err := r.Run(":" + os.Getenv("APP_PORT"))
	if err != nil {
		log.Println("服务启动异常：", err)
		return
	}
	log.Println("服务已成功启动.")

}
