package models

import (
	"log"
	"os"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 数据库驱动名
const (
	DriverMySQL    = "mysql"
	DriverPostgres = "postgres"
)

var DB *gorm.DB

func init() {
	InitDB()
	InitCasbin()
}

// ResolveDriver 返回驱动名。优先取环境变量 DB_DRIVER（mysql / postgres），
// 未配置时按 DSN 形式推断，保证既有 MySQL 部署无需改动即可运行。
func ResolveDriver(driver, dsn string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case DriverPostgres, "postgresql", "pg":
		return DriverPostgres
	case DriverMySQL, "mariadb":
		return DriverMySQL
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") ||
		strings.Contains(dsn, "sslmode=") || strings.Contains(dsn, "fallback_application_name=") {
		return DriverPostgres
	}
	return DriverMySQL
}

// Open 按 DSN 形式打开连接，MySQL 与 PostgreSQL 共用同一套 GORM 模型与 AutoMigrate 流程。
func Open(dsn string) (*gorm.DB, error) {
	cfg := &gorm.Config{
		SkipDefaultTransaction: true,
	}
	driver := ResolveDriver(os.Getenv("DB_DRIVER"), dsn)
	if driver == DriverPostgres {
		return gorm.Open(newPostgresDialector(dsn), cfg)
	}
	return gorm.Open(mysql.Open(dsn), cfg)
}

func InitDB() {
	// DB_AUTOINIT=off 时跳过连接与迁移，供 cmd/migrate2pg 等工具复用模型定义而不触碰数据库
	if strings.EqualFold(os.Getenv("DB_AUTOINIT"), "off") {
		return
	}
	dsn := os.Getenv("DB_DSN")
	db, err := Open(dsn)
	if err != nil {
		log.Println("Failed to connect to database, Err:", err)
		return
	}
	// 自动执行表迁移操作
	tables := MigrateTables()

	for _, table := range tables {
		if err = db.AutoMigrate(table); err != nil {
			log.Println("Failed to auto migrate table, Err:", err)
			return
		}
	}
	DB = db
}

// MigrateTables 返回需要 AutoMigrate 的全部模型，供启动流程与建库工具共用。
func MigrateTables() []interface{} {
	return []interface{}{
		&SysUser{},
		&SysLog{},
		&SysLoginLog{},
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
		&OpsScript{},
		&OpsMonitorTarget{},
		&OpsMonitorRule{},
		&OpsMonitorAlertEvent{},
		&OpsMonitorNotifyChannel{},
		&OpsMonitorNotifyLog{},
		&OpsLogSource{},
		&OpsLogAlertRule{},
		&OpsCmdbAsset{},
		&OpsCmdbTag{},
		&OpsCmdbAssetTag{},
		&OpsCmdbRelation{},
		&OpsCmdbChangeLog{},
		&OpsCmdbSubnet{},
		&OpsSessionRecord{},
	}
}

func InitCasbin() {
	Casbin = &CasbinHandler{}
	// 初始化将在第一次使用时自动完成
	// 可以通过 Casbin.IsInitialized() 检查状态
	// 通过 Casbin.GetInitError() 获取错误信息
}
