package main

import (
	"github.com/zhany/ops-go/bastion"
	"github.com/zhany/ops-go/routers"
	"log"
	"os"
)

func main() {
	// 启动堡垒机服务
	go bastion.Init()
	// 启动堡垒机
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
