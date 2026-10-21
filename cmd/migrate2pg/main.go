// MySQL → PostgreSQL 一次性停机迁移工具。
//
// 必须在 ops-go 目录下、并带上 DB_AUTOINIT=off 运行，否则 models 包会按 .env 自动连接并
// 对源库执行 AutoMigrate：
//
//	go run ./cmd/migrate2pg -pg "host=1.2.3.4 port=5432 user=postgres password=secret dbname=ops_go sslmode=disable timezone=Asia/Shanghai" -action schema
//	go run ./cmd/migrate2pg -pg "..." -action check
//	go run ./cmd/migrate2pg -pg "..." -action copy
//	go run ./cmd/migrate2pg -pg "..." -action verify
//	go run ./cmd/migrate2pg -pg "..." -action sequences   # 仅重置序列（copy 中断后可单独补跑）
//
// 源库 DSN 默认取 .env 的 DB_DSN（需含 parseTime=True），也可用 -mysql 覆盖。
// 本工具只读源库，不写入 MySQL。
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/zhany/ops-go/models"
)

const batchSize = 1000

// tablePlan 一张表的迁移计划：列名 + PG 侧类型 + 主键生成方式
type tablePlan struct {
	name     string
	columns  []string
	pgTypes  map[string]string
	identity map[string]string // 列名 → attidentity（'a' 强制 / 'd' 默认）
	nulCols  map[string]int    // 清洗过 NUL 字节的列及次数
	rows     int64
}

func main() {
	pgDSN := flag.String("pg", "", "PostgreSQL 目标 DSN（key=value 或 postgres:// 形式）")
	mysqlDSN := flag.String("mysql", "", "MySQL 源 DSN，默认读取 .env 的 DB_DSN")
	action := flag.String("action", "check", "schema=在 PG 建表 / check=行数与列差异盘点 / copy=搬数据 / verify=复核行数 / sequences=重置自增序列")
	only := flag.String("tables", "", "仅处理指定表（逗号分隔），默认全部")
	skipTruncate := flag.Bool("keep", false, "copy 时不清空目标表（默认先 TRUNCATE ... RESTART IDENTITY CASCADE）")
	flag.Parse()

	if *pgDSN == "" {
		fatal("-pg 目标 DSN 不能为空")
	}
	if models.DB != nil {
		fatal("检测到 models 包已自动连接数据库，请带 DB_AUTOINIT=off 运行本工具")
	}
	if err := godotenv.Load(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "提示: 未读取到 .env:", err)
	}
	src := *mysqlDSN
	if src == "" {
		src = os.Getenv("DB_DSN")
	}
	if src == "" {
		fatal(".env 缺少 DB_DSN，且未提供 -mysql")
	}
	if models.ResolveDriver("", src) != models.DriverMySQL {
		fatal("源库 DSN 看起来不是 MySQL，请用 -mysql 显式指定")
	}
	if !strings.Contains(src, "parseTime=True") {
		fmt.Fprintln(os.Stderr, "提示: 源库 DSN 未含 parseTime=True，时间列将以字符串搬运，零日期可能报错")
	}

	mysqlDB, err := sql.Open("mysql", src)
	if err != nil {
		fatal("打开 MySQL 失败: ", err)
	}
	defer mysqlDB.Close()
	if err = mysqlDB.Ping(); err != nil {
		fatal("连接 MySQL 失败: ", err)
	}

	switch *action {
	case "schema":
		closeMysql(mysqlDB)
		createSchema(*pgDSN)
		return
	case "check", "copy", "verify", "sequences":
	default:
		fatal("未知 -action: ", *action)
	}

	pgDB, err := sql.Open("pgx", *pgDSN)
	if err != nil {
		fatal("打开 PostgreSQL 失败: ", err)
	}
	defer pgDB.Close()
	if err = pgDB.Ping(); err != nil {
		fatal("连接 PostgreSQL 失败: ", err)
	}

	srcTables, err := mysqlTableNames(mysqlDB)
	if err != nil {
		fatal("读取 MySQL 表清单失败: ", err)
	}
	pgTables, err := pgTableNames(pgDB)
	if err != nil {
		fatal("读取 PostgreSQL 表清单失败: ", err)
	}
	if len(pgTables) == 0 {
		fatal("目标库没有任何表，请先执行 -action schema")
	}

	var plans []*tablePlan
	for _, t := range srcTables {
		if !include(*only, t) {
			continue
		}
		if !pgTables[t] {
			fmt.Printf("%-32s 跳过：目标库缺少该表\n", t)
			continue
		}
		plan, err := buildPlan(mysqlDB, pgDB, t)
		if err != nil {
			fatal(fmt.Sprintf("准备 %s 失败: ", t), err)
		}
		plans = append(plans, plan)
	}
	plans = sortByFK(pgDB, plans)

	switch *action {
	case "check":
		reportCheck(plans)
	case "verify":
		reportVerify(pgDB, plans)
	case "copy":
		copyAll(pgDB, mysqlDB, plans, *skipTruncate)
	case "sequences":
		if err := resetSequences(pgDB, plans); err != nil {
			fatal("重置序列失败: ", err)
		}
		fmt.Println("序列已重置")
	}
}

func closeMysql(db *sql.DB) { _ = db.Close() }

func fatal(v ...any) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func include(only, table string) bool {
	only = strings.TrimSpace(only)
	if only == "" {
		return true
	}
	for _, t := range strings.Split(only, ",") {
		if strings.TrimSpace(t) == table {
			return true
		}
	}
	return false
}

// createSchema 用与启动流程完全相同的模型列表在 PG 建表，确保 DDL 与代码一致
func createSchema(pgDSN string) {
	db, err := models.Open(pgDSN)
	if err != nil {
		fatal("连接 PostgreSQL 失败: ", err)
	}
	models.DB = db
	for _, table := range models.MigrateTables() {
		if err = db.AutoMigrate(table); err != nil {
			fatal("AutoMigrate 建表失败: ", err)
		}
	}
	// casbin_rule 由 casbin adapter 建表
	if _, err = newCasbinAdapter(db); err != nil {
		fatal("创建 casbin_rule 失败: ", err)
	}
	fmt.Println("PostgreSQL 结构创建完成（GORM AutoMigrate + casbin_rule）")
}

func mysqlTableNames(db *sql.DB) ([]string, error) {
	var name string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&name); err != nil {
		return nil, err
	}
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND table_type = 'BASE TABLE' ORDER BY table_name", name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		if err = rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func pgTableNames(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = current_schema() AND c.relkind = 'r'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

// buildPlan 取 MySQL 列与 PG 列的交集，PG 类型决定绑定值的转换方式
func buildPlan(src, dst *sql.DB, table string) (*tablePlan, error) {
	plan := &tablePlan{
		name:     table,
		pgTypes:  map[string]string{},
		identity: map[string]string{},
		nulCols:  map[string]int{},
	}
	srcCols, err := mysqlColumns(src, table)
	if err != nil {
		return nil, err
	}
	pgCols, pgTypes, identity, err := pgColumns(dst, table)
	if err != nil {
		return nil, err
	}
	inPG := map[string]bool{}
	for _, c := range pgCols {
		inPG[c] = true
	}
	for _, c := range srcCols {
		if !inPG[c] {
			fmt.Printf("%-32s 警告: 源库列 %s 在目标库不存在，该列不迁移\n", table, c)
			continue
		}
		plan.columns = append(plan.columns, c)
		plan.pgTypes[c] = pgTypes[c]
		if identity[c] != "" {
			plan.identity[c] = identity[c]
		}
	}
	if err = src.QueryRow("SELECT COUNT(*) FROM " + mysqlQuote(table)).Scan(&plan.rows); err != nil {
		return nil, err
	}
	return plan, nil
}

func mysqlColumns(db *sql.DB, table string) ([]string, error) {
	var schema string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&schema); err != nil {
		return nil, err
	}
	rows, err := db.Query("SELECT column_name FROM information_schema.columns WHERE table_schema = ? AND table_name = ? ORDER BY ordinal_position", schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err = rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func pgColumns(db *sql.DB, table string) (names []string, types map[string]string, identity map[string]string, err error) {
	rows, err := db.Query(`SELECT a.attname, a.atttypid::regtype::text, a.attidentity
FROM pg_attribute a JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = current_schema() AND c.relname = $1 AND a.attnum > 0 AND NOT a.attisdropped
ORDER BY a.attnum`, table)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	types, identity = map[string]string{}, map[string]string{}
	for rows.Next() {
		var name, typ string
		var idt sql.NullString
		if err = rows.Scan(&name, &typ, &idt); err != nil {
			return nil, nil, nil, err
		}
		names = append(names, name)
		types[name] = typ
		identity[name] = idt.String
	}
	return names, types, identity, rows.Err()
}

// sortByFK 按外键依赖排序，被引用表先写，避免单表重跑时的约束冲突
func sortByFK(db *sql.DB, plans []*tablePlan) []*tablePlan {
	byName := map[string]*tablePlan{}
	for _, p := range plans {
		byName[p.name] = p
	}
	deps := map[string][]string{}
	rows, err := db.Query(`SELECT c.conrelid::regclass::text, c.confrelid::regclass::text
FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace
WHERE n.nspname = current_schema() AND c.contype = 'f'`)
	if err == nil {
		for rows.Next() {
			var child, parent string
			if scanErr := rows.Scan(&child, &parent); scanErr == nil && child != parent {
				deps[child] = append(deps[child], parent)
			}
		}
		rows.Close()
	}
	var out []*tablePlan
	placed := map[string]bool{}
	for len(out) < len(plans) {
		progress := false
		for _, p := range plans {
			if placed[p.name] {
				continue
			}
			ready := true
			for _, d := range deps[p.name] {
				if byName[d] != nil && !placed[d] {
					ready = false
					break
				}
			}
			if ready {
				out = append(out, p)
				placed[p.name] = true
				progress = true
			}
		}
		if !progress { // 外键成环，退回原顺序
			for _, p := range plans {
				if !placed[p.name] {
					out = append(out, p)
					placed[p.name] = true
				}
			}
		}
	}
	return out
}

func reportCheck(plans []*tablePlan) {
	var total int64
	fmt.Printf("%-32s %10s %6s\n", "TABLE", "MYSQL_ROWS", "COLUMNS")
	for _, p := range plans {
		total += p.rows
		fmt.Printf("%-32s %10d %6d\n", p.name, p.rows, len(p.columns))
	}
	fmt.Printf("合计 %d 张表 / %d 行\n", len(plans), total)
}

func reportVerify(dst *sql.DB, plans []*tablePlan) {
	fmt.Printf("%-32s %10s %10s %s\n", "TABLE", "MYSQL", "POSTGRES", "RESULT")
	var diff int
	for _, p := range plans {
		var dstRows int64
		if err := dst.QueryRow("SELECT COUNT(*) FROM " + pgQuote(p.name)).Scan(&dstRows); err != nil {
			fmt.Printf("%-32s %10d %10s 查询失败: %v\n", p.name, p.rows, "-", err)
			diff++
			continue
		}
		status := "OK"
		if dstRows != p.rows {
			status = "不一致"
			diff++
		}
		fmt.Printf("%-32s %10d %10d %s\n", p.name, p.rows, dstRows, status)
	}
	fmt.Printf("%d 张表存在差异\n", diff)
	if diff > 0 {
		os.Exit(1)
	}
}

func copyAll(dst, src *sql.DB, plans []*tablePlan, keep bool) {
	for _, p := range plans {
		if err := copyTable(dst, src, p, keep); err != nil {
			fatal(fmt.Sprintf("迁移 %s 失败: ", p.name), err)
		}
	}
	if err := resetSequences(dst, plans); err != nil {
		fatal("重置序列失败: ", err)
	}
	fmt.Println("数据迁移完成，执行 -action verify 复核行数")
}

func copyTable(dst, src *sql.DB, p *tablePlan, keep bool) error {
	started := time.Now()
	if !keep {
		if _, err := dst.Exec("TRUNCATE " + pgQuote(p.name) + " RESTART IDENTITY CASCADE"); err != nil {
			return err
		}
	}
	cols := make([]string, len(p.columns))
	for i, c := range p.columns {
		cols[i] = pgQuote(c)
	}
	override := ""
	for _, c := range p.columns {
		if p.identity[c] == "a" {
			override = " OVERRIDING SYSTEM VALUE"
		}
	}
	prefix := fmt.Sprintf("INSERT INTO %s (%s) VALUES ", pgQuote(p.name), strings.Join(cols, ", "))

	rows, err := src.Query("SELECT " + strings.Join(mapCols(p.columns, mysqlQuote), ", ") + " FROM " + mysqlQuote(p.name))
	if err != nil {
		return err
	}
	defer rows.Close()

	var copied int64
	buf := make([]any, len(p.columns))
	ptrs := make([]any, len(p.columns))
	for i := range buf {
		ptrs[i] = &buf[i]
	}
	var args []any
	var pending []string
	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		_, err := dst.Exec(prefix+strings.Join(pending, ", ")+override, args...)
		pending, args = pending[:0], args[:0]
		return err
	}
	for rows.Next() {
		if err = rows.Scan(ptrs...); err != nil {
			return err
		}
		ph := make([]string, len(p.columns))
		for i, c := range p.columns {
			v, convErr := p.convert(c, buf[i], p.pgTypes[c])
			if convErr != nil {
				return fmt.Errorf("列 %s: %w", c, convErr)
			}
			args = append(args, v)
			ph[i] = "$" + strconv.Itoa(len(args))
		}
		pending = append(pending, "("+strings.Join(ph, ", ")+")")
		copied++
		if len(pending) >= batchSize {
			if err = flush(); err != nil {
				return err
			}
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if err = flush(); err != nil {
		return err
	}
	if copied < p.rows {
		return fmt.Errorf("写入 %d 行，少于源库快照 %d 行", copied, p.rows)
	}
	if copied > p.rows {
		fmt.Printf("%-32s 提示: 搬迁期间源库新增 %d 行（快照 %d，实搬 %d），停机后请重跑本表\n",
			p.name, copied-p.rows, p.rows, copied)
	}
	if len(p.nulCols) > 0 {
		for col, n := range p.nulCols {
			fmt.Printf("%-32s 警告: 列 %s 清洗了 %d 处 NUL 字节\n", p.name, col, n)
		}
	}
	fmt.Printf("%-32s %10d 行  %.1fs\n", p.name, copied, time.Since(started).Seconds())
	return nil
}

func mapCols(cols []string, quote func(string) string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = quote(c)
	}
	return out
}

func resetSequences(dst *sql.DB, plans []*tablePlan) error {
	for _, p := range plans {
		for _, c := range p.columns {
			if p.identity[c] == "" && !isSerial(dst, p.name, c) {
				continue
			}
			var maxID int64
			if err := dst.QueryRow(fmt.Sprintf("SELECT COALESCE(MAX(%s), -1) FROM %s", pgQuote(c), pgQuote(p.name))).Scan(&maxID); err != nil {
				return err
			}
			called := false
			// is_called=false 表示下一个取值就是这个值本身，因此传 max+1（空表传 1）
			next := maxID + 1
			if next < 1 {
				next = 1
			}
			if _, err := dst.Exec("SELECT setval(pg_get_serial_sequence($1, $2), $3, $4)", p.name, c, next, called); err != nil {
				return fmt.Errorf("%s.%s: %w", p.name, c, err)
			}
		}
	}
	return nil
}

func isSerial(db *sql.DB, table, column string) bool {
	var n int
	err := db.QueryRow(`SELECT count(*) FROM pg_attribute a JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace ns ON ns.oid = c.relnamespace
WHERE ns.nspname = current_schema() AND c.relname = $1 AND a.attname = $2
AND (a.attidentity <> '' OR pg_get_serial_sequence(c.oid::regclass::text, a.attname) IS NOT NULL)`, table, column).Scan(&n)
	return err == nil && n > 0
}

// convert 按目标列类型转换 MySQL 驱动返回值（数值/字符多为 []byte）
func (p *tablePlan) convert(col string, v any, pgType string) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch pgType {
	case "smallint", "integer", "bigint":
		return toInt(v)
	case "numeric", "double precision", "real":
		return toFloat(v)
	case "boolean":
		return toBool(v)
	case "timestamp", "timestamp with time zone":
		return toTime(v)
	case "date":
		t, err := toTime(v)
		if err != nil || t == nil {
			return nil, err
		}
		return t.(time.Time).Format("2006-01-02"), nil
	default: // text / character varying / character / json / jsonb / uuid ...
		switch x := v.(type) {
		case []byte:
			return p.dropNul(col, string(x))
		case string:
			return p.dropNul(col, x)
		default:
			return v, nil
		}
	}
}

// dropNul 去掉 NUL 字节：MySQL 的字符串列允许 0x00，PostgreSQL 直接拒绝整条语句
func (p *tablePlan) dropNul(col, s string) (string, error) {
	if !strings.ContainsRune(s, 0) {
		return s, nil
	}
	p.nulCols[col]++
	return strings.ReplaceAll(s, "\x00", ""), nil
}

func toInt(v any) (int64, error) {
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int8:
		return int64(x), nil
	case int16:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case uint:
		return int64(x), nil
	case uint8:
		return int64(x), nil
	case uint16:
		return int64(x), nil
	case uint32:
		return int64(x), nil
	case uint64:
		if x > math.MaxInt64 {
			return 0, fmt.Errorf("BIGINT UNSIGNED 值 %d 超出 int64", x)
		}
		return int64(x), nil
	case int64:
		return x, nil
	case float64:
		return int64(x), nil
	case []byte:
		return strconv.ParseInt(strings.TrimSpace(string(x)), 10, 64)
	case string:
		return strconv.ParseInt(strings.TrimSpace(x), 10, 64)
	default:
		return 0, fmt.Errorf("无法转成整数: %T", v)
	}
}

func toFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		n, err := toInt(x)
		return float64(n), err
	case []byte:
		return strconv.ParseFloat(strings.TrimSpace(string(x)), 64)
	case string:
		return strconv.ParseFloat(strings.TrimSpace(x), 64)
	default:
		return 0, fmt.Errorf("无法转成数值: %T", v)
	}
}

func toBool(v any) (bool, error) {
	switch x := v.(type) {
	case bool:
		return x, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		n, err := toInt(x)
		return n != 0, err
	case []byte:
		return parseBool(string(x))
	case string:
		return parseBool(x)
	default:
		return false, fmt.Errorf("无法转成布尔: %T", v)
	}
}

func parseBool(s string) (bool, error) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "1", "t", "true", "y", "yes":
		return true, nil
	case "0", "f", "false", "n", "no", "":
		return false, nil
	default:
		return false, fmt.Errorf("无法转成布尔: %q", s)
	}
}

var timeLayouts = []string{"2006-01-02 15:04:05.999999", "2006-01-02 15:04:05", "2006-01-02"}

func toTime(v any) (any, error) {
	switch x := v.(type) {
	case time.Time:
		if x.Year() < 1000 { // MySQL 零日期 0000-00-00
			return nil, nil
		}
		return x, nil
	case []byte:
		return parseTime(string(x))
	case string:
		return parseTime(x)
	default:
		return nil, fmt.Errorf("无法转成时间: %T", v)
	}
}

func parseTime(s string) (any, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "0000-00-00") {
		return nil, nil
	}
	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return nil, fmt.Errorf("无法解析时间: %q", s)
}

func mysqlQuote(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func pgQuote(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
