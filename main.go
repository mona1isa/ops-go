package main

import (
	"log"
	"os"

	"github.com/zhany/ops-go/bastion"
	"github.com/zhany/ops-go/routers"
)

func main() {
	// 启动堡垒机服务
	go bastion.Init()

	r := routers.Init()
	err := r.Run(":" + os.Getenv("APP_PORT"))
	if err != nil {
		log.Println("服务启动异常：", err)
		return
	}
	log.Println("服务已成功启动.")
}
