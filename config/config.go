package config

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/zhany/ops-go/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
	"strconv"
)

var DB *gorm.DB
var RedisClient *redis.Client
var ctx = context.Background()

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file, Err:", err)
	}
}

func InitDB() {
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println("Failed to connect to database, Err:", err)
	}
	// 自动执行表迁移操作
	tables := []interface{}{
		&models.SysUser{},
		&models.SysLog{},
		&models.SysRole{},
		&models.SysUserRole{},
		&models.SysMenu{},
		&models.SysRoleMenu{},
		&models.SysUserToken{},
		&models.SysDept{},
	}

	for _, table := range tables {
		if err = db.AutoMigrate(table); err != nil {
			log.Println("Failed to auto migrate table, Err:", err)
			return
		}
	}
	DB = db
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
