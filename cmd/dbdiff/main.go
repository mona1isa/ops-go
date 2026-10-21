package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

// 成对查询：左 MySQL，右 PostgreSQL，结果应一致（除已知的 NUL 清洗）。
// 注意 MySQL LENGTH() 返回字节数、PG length() 返回字符数，中文列必须用 CHAR_LENGTH 对齐。
var pairs = [][2]string{
	{"SELECT count(*), min(id), max(id), sum(CHAR_LENGTH(password)), sum(CHAR_LENGTH(nick_name)) FROM sys_user",
		"SELECT count(*), min(id), max(id), sum(length(password)), coalesce(sum(length(nick_name)),0) FROM sys_user"},
	{"SELECT count(*), sum(status = '1'), sum(del_flag = '0') FROM sys_user",
		"SELECT count(*), sum((status = '1')::int), sum((del_flag = '0')::int) FROM sys_user"},
	// 逐行 md5 前 8 位转数值再求和，等价于整表内容指纹
	{"SELECT CAST(sum(CONV(LEFT(MD5(CONCAT_WS('|', id, user_name, nick_name, email, phone, status, del_flag)), 8), 16, 10)) AS CHAR) FROM sys_user",
		"SELECT sum(('x'||substr(md5(concat_ws('|', id::text, user_name, nick_name, email, phone, status::text, del_flag)),1,8))::bit(32)::bigint) FROM sys_user"},
	{"SELECT count(*) FROM sys_menu WHERE status = '1'", "SELECT count(*) FROM sys_menu WHERE status"},
	{"SELECT count(*) FROM sys_menu WHERE status = '0'", "SELECT count(*) FROM sys_menu WHERE NOT status"},
	{"SELECT count(*), sum(keep_alive), sum(is_affix), sum(is_hide), sum(is_iframe), sum(is_link) FROM sys_menu",
		"SELECT count(*), sum(keep_alive::int), sum(is_affix::int), sum(is_hide::int), sum(is_iframe::int), sum(is_link::int) FROM sys_menu"},
	{"SELECT count(*), sum(CHAR_LENGTH(name)), sum(CHAR_LENGTH(perms)) FROM sys_menu",
		"SELECT count(*), coalesce(sum(length(name)),0), coalesce(sum(length(perms)),0) FROM sys_menu"},
	{"SELECT CAST(sum(CONV(LEFT(MD5(CONCAT_WS('|', id, name, parent_id, order_num, path, type, perms)), 8), 16, 10)) AS CHAR) FROM sys_menu",
		"SELECT sum(('x'||substr(md5(concat_ws('|', id::text, name, parent_id::text, order_num::text, path, type, perms)),1,8))::bit(32)::bigint) FROM sys_menu"},
	{"SELECT count(*), sum(order_num), sum(CHAR_LENGTH(name)) FROM sys_role",
		"SELECT count(*), sum(order_num), sum(length(name)) FROM sys_role"},
	{"SELECT count(*) FROM ops_instance WHERE status = '1' AND del_flag = '0'",
		"SELECT count(*) FROM ops_instance WHERE status = '1' AND del_flag = '0'"},
	{"SELECT count(*), sum(CHAR_LENGTH(name)), sum(CHAR_LENGTH(ip)), sum(cpu), sum(mem_mb), sum(disk_gb) FROM ops_instance",
		"SELECT count(*), coalesce(sum(length(name)),0), coalesce(sum(length(ip)),0), sum(cpu::bigint), sum(mem_mb::bigint), sum(disk_gb::bigint) FROM ops_instance"},
	{"SELECT CAST(sum(CONV(LEFT(MD5(CONCAT_WS('|', id, name, ip, os, status, cpu, mem_mb)), 8), 16, 10)) AS CHAR) FROM ops_instance",
		"SELECT sum(('x'||substr(md5(concat_ws('|', id::text, name, ip, os, status, cpu::text, mem_mb::text)),1,8))::bit(32)::bigint) FROM ops_instance"},
	{"SELECT count(*), sum(del_flag = '0'), sum(del_flag = '1') FROM ops_user_instance_auth",
		"SELECT count(*), sum((del_flag = '0')::int), sum((del_flag = '1')::int) FROM ops_user_instance_auth"},
	{"SELECT count(*), sum(role_id), sum(menu_id) FROM sys_role_menu",
		"SELECT count(*), sum(role_id), sum(menu_id) FROM sys_role_menu"},
	{"SELECT count(*), sum(user_id), sum(instance_id), sum(key_id) FROM ops_user_instance_key_auth",
		"SELECT count(*), sum(user_id), sum(instance_id), sum(key_id) FROM ops_user_instance_key_auth"},
	{"SELECT min(created_at), max(created_at), count(*), sum(result = 1), sum(CHAR_LENGTH(user_name)) FROM sys_login_log",
		"SELECT min(created_at), max(created_at), count(*), sum((result = 1)::int), sum(length(user_name)) FROM sys_login_log"},
	{"SELECT min(created_at), max(created_at) FROM ops_session_record",
		"SELECT min(created_at), max(created_at) FROM ops_session_record"},
	{"SELECT count(*), sum(CHAR_LENGTH(params)), sum(cost_time_ms) FROM sys_log",
		"SELECT count(*), coalesce(sum(length(params)),0), coalesce(sum(cost_time_ms),0) FROM sys_log"},
	{"SELECT count(*), sum(interval_sec), sum(timeout_sec), sum(last_status = 1) FROM ops_monitor_target",
		"SELECT count(*), sum(interval_sec), sum(timeout_sec), sum((last_status = 1)::int) FROM ops_monitor_target"},
	{"SELECT count(*) FROM casbin_rule", "SELECT count(*) FROM casbin_rule"},
	{"SELECT count(*), sum(v0 <> '') FROM casbin_rule", "SELECT count(*), sum((v0 <> '')::int) FROM casbin_rule"},
	{"SELECT CAST(sum(CONV(LEFT(MD5(CONCAT_WS('|', id, ptype, v0, v1, v2, v3, v4, v5)), 8), 16, 10)) AS CHAR) FROM casbin_rule",
		"SELECT sum(('x'||substr(md5(concat_ws('|', id::text, ptype, v0, v1, v2, v3, v4, v5)),1,8))::bit(32)::bigint) FROM casbin_rule"},
	{"SELECT count(*) FROM ops_cmdb_asset WHERE status = 3", "SELECT count(*) FROM ops_cmdb_asset WHERE status = 3"},
	{"SELECT count(*), sum(CHAR_LENGTH(content)) FROM ops_script",
		"SELECT count(*), coalesce(sum(length(content)),0) FROM ops_script"},
	{"SELECT count(*), sum(CHAR_LENGTH(pattern)) FROM ops_dangerous_command",
		"SELECT count(*), coalesce(sum(length(pattern)),0) FROM ops_dangerous_command"},
}

func main() {
	godotenv.Load(".env")
	mydsn := os.Getenv("DB_DSN")
	pgdsn := os.Getenv("PG_DSN")
	if pgdsn == "" {
		fmt.Println("需要 PG_DSN 环境变量")
		os.Exit(1)
	}
	my, err := sql.Open("mysql", mydsn)
	if err != nil {
		panic(err)
	}
	defer my.Close()
	pg, err := sql.Open("pgx", pgdsn)
	if err != nil {
		panic(err)
	}
	defer pg.Close()

	bad := 0
	for _, p := range pairs {
		a := one(my, p[0])
		b := one(pg, p[1])
		ok := a == b
		if !ok {
			bad++
		}
		status := "OK"
		if !ok {
			status = "不一致"
		}
		fmt.Printf("%-8s MySQL=%-60s PG=%-60s %s\n", status, trim(a), trim(b), p[1][:min(40, len(p[1]))])
	}
	fmt.Println("不一致项:", bad)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func one(db *sql.DB, q string) string {
	rows, err := db.Query(q)
	if err != nil {
		return "ERR: " + err.Error()
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return "ERR: " + err.Error()
		}
		for i, v := range vals {
			if i > 0 {
				out += "|"
			}
			switch x := v.(type) {
			case []byte:
				out += string(x)
			case nil:
				out += "NULL"
			default:
				out += fmt.Sprint(x)
			}
		}
	}
	return out
}

func trim(s string) string {
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}
