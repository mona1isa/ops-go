package config

import (
	"github.com/joho/godotenv"
	"github.com/zhany/ops-go/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
)

var DB *gorm.DB

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
	if err = db.AutoMigrate(&models.SysUser{}, &models.SysLog{}); err != nil {
		log.Println("Failed to auto migrate DB, Err:", err)
	}
	DB = db
}
