package models

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
)

var DB *gorm.DB

func init() {
	InitDB()
	InitCasbin()
}

func InitDB() {
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		log.Println("Failed to connect to database, Err:", err)
	}
	// 自动执行表迁移操作
	tables := []interface{}{
		&SysUser{},
		&SysLog{},
		&SysRole{},
		&SysUserRole{},
		&SysMenu{},
		&SysRoleMenu{},
		&SysUserToken{},
		&SysDept{},
		&OpsInstance{},
		&OpsKey{},
		&OpsInstanceKey{},
		&OpsGroup{},
		&OpsInstanceGroup{},
		&OpsUserInstanceAuth{},
		&OpsUserInstanceKeyAuth{},
		&OpsDangerousCommand{},
		&OpsTaskTemplate{},
		&OpsTaskPipeline{},
		&OpsPipelineStep{},
		&OpsTaskExecution{},
		&OpsExecutionHost{},
		&OpsStepExecution{},
	}

	for _, table := range tables {
		if err = db.AutoMigrate(table); err != nil {
			log.Println("Failed to auto migrate table, Err:", err)
			return
		}
	}
	DB = db
}

func InitCasbin() {
	Casbin = &CasbinHandler{}
	// 初始化将在第一次使用时自动完成
	// 可以通过 Casbin.IsInitialized() 检查状态
	// 通过 Casbin.GetInitError() 获取错误信息
}
