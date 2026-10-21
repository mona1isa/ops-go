package models

import "fmt"

// DialectName 返回当前连接的数据库方言名（mysql / postgres）。
func DialectName() string {
	if DB == nil {
		return DriverMySQL
	}
	return DB.Dialector.Name()
}

// IsPostgres 判断当前连接是否为 PostgreSQL。
func IsPostgres() bool {
	return DialectName() == DriverPostgres
}

// DayExpr 返回按天截取时间列的可移植表达式片段，用于分组统计。
// column 必须是代码内的字面量列名，不接受外部输入。
func DayExpr(column string) string {
	if IsPostgres() {
		return fmt.Sprintf("to_char(%s, 'YYYY-MM-DD')", column)
	}
	return fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m-%%d')", column)
}

// QuotedIdent 按当前方言给表名/列名加引用符（仅用于代码内字面量标识符）。
func QuotedIdent(name string) string {
	if IsPostgres() {
		return `"` + name + `"`
	}
	return "`" + name + "`"
}
