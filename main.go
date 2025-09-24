package main

import (
	"github.com/zhany/ops-go/routers"
	"log"
	"os"
)

func main() {
	r := routers.Init()
	err := r.Run(":" + os.Getenv("APP_PORT"))
	if err != nil {
		log.Println("服务启动异常：", err)
		return
	}
	log.Println("服务已成功启动.")
}
