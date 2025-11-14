package config

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/zhany/ops-go/bastion"
	"log"
	"os"
	"strconv"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

func init() {
	LoadEnv()
	InitRedis()
	go bastion.Init()
}

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file, Err:", err)
	}
}

func InitRedis() {
	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	redisPoolSize, _ := strconv.Atoi(os.Getenv("REDIS_POOL_SIZE"))
	minIdleConn, _ := strconv.Atoi(os.Getenv("REDIS_MIN_IDLE_CONN"))
	client := redis.NewClient(&redis.Options{
		Addr:         os.Getenv("REDIS_ADDRESS"), // Redis地址
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           redisDB, // 默认数据库
		PoolSize:     redisPoolSize,
		MinIdleConns: minIdleConn,
	})
	// 测试连接性
	if _, err := client.Ping(ctx).Result(); err != nil {
		log.Println("Failed to connect to Redis, Err:", err)
		return
	}
	RedisClient = client
}
