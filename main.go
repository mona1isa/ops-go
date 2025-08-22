package main

import (
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/routers"
	"log"
	"os"
)

func main() {
	config.LoadEnv()
	config.InitDB()

	r := routers.Init()
	err := r.Run(":" + os.Getenv("APP_PORT"))
	if err != nil {
		log.Println("服务启动异常：", err)
		return
	}
}
