// 从当前运行库导出全量 DDL，用于一键初始化（不含数据）。
//
// 用法（在 ops-go 目录下，读取 .env 的 DB_DSN）：
//   go run ./cmd/sqldump > ../sql/ops-go.sql
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// dbName 从 DSN 中解析库名：user:pass@tcp(host:port)/dbname?params
func dbName(dsn string) string {
	at := strings.LastIndex(dsn, "@")
	if at < 0 {
		return ""
	}
	rest := dsn[at+1:]
	slash := strings.Index(rest, "/")
	if slash < 0 {
		return ""
	}
	name := rest[slash+1:]
	if q := strings.IndexAny(name, "?"); q >= 0 {
		name = name[:q]
	}
	return name
}

func main() {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "读取 .env 失败（请在 ops-go 目录下执行）:", err)
		os.Exit(1)
	}
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, ".env 缺少 DB_DSN")
		os.Exit(1)
	}
	dbName := dbName(dsn)
	if dbName == "" {
		fmt.Fprintln(os.Stderr, "无法从 DB_DSN 解析库名")
		os.Exit(1)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "连接数据库失败:", err)
		os.Exit(1)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "连接数据库失败:", err)
		os.Exit(1)
	}

	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = ? ORDER BY table_name", dbName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "查询表清单失败:", err)
		os.Exit(1)
	}
	var tables []string
	for rows.Next() {
		var t string
		if err = rows.Scan(&t); err != nil {
			rows.Close()
			fmt.Fprintln(os.Stderr, "读取表名失败:", err)
			os.Exit(1)
		}
		tables = append(tables, t)
	}
	rows.Close()

	out := os.Stdout
	fmt.Fprintf(out, "-- Ops-Go 数据库初始化 DDL（由 cmd/sqldump 从运行库导出，共 %d 张表）\n", len(tables))
	fmt.Fprintf(out, "-- 生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(out, "-- 执行: mysql -u<user> -p < ops-go.sql\n")
	fmt.Fprintf(out, "-- 说明: 应用启动时 GORM AutoMigrate 也会建表，本文件供全新环境一键初始化\n\n")
	fmt.Fprintf(out, "CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;\n", dbName)
	fmt.Fprintf(out, "USE `%s`;\n\n", dbName)

	for _, table := range tables {
		var name, create string
		err = db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", dbName, table)).Scan(&name, &create)
		if err != nil {
			fmt.Fprintf(os.Stderr, "导出 %s 失败: %v\n", table, err)
			os.Exit(1)
		}
		fmt.Fprintf(out, "DROP TABLE IF EXISTS `%s`;\n%s;\n\n", table, create)
	}
}
