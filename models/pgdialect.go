package models

import (
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// pgDialector 在 PostgreSQL 上把模型里面向 MySQL 的 type 标签翻译成等价的 PG 类型。
//
// 项目的 gorm 标签长期按 MySQL 书写（int(11)、tinyint(4)、char(1) 存布尔等），
// 这些在 PG 里要么是语法错误，要么与 Go 字段类型冲突。为免同时改动两种方言的
// 建表结果，统一在这一处适配，模型定义保持原样。
type pgDialector struct {
	*postgres.Dialector
}

func newPostgresDialector(dsn string) gorm.Dialector {
	return pgDialector{&postgres.Dialector{Config: &postgres.Config{DSN: dsn}}}
}

// Migrator 必须重建，让迁移器持有本包装而不是内层 dialector，否则 DataTypeOf 的改写不生效。
func (d pgDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return postgres.Migrator{Migrator: migrator.Migrator{Config: migrator.Config{
		DB:                          db,
		Dialector:                   d,
		CreateIndexAfterCreateTable: true,
	}}}
}

func (d pgDialector) DataTypeOf(field *schema.Field) string {
	// 布尔字段一律用 PG 原生 boolean：char(1)/tinyint 上的 true/false 写入会被 PG 拒绝
	if field.GORMDataType == schema.Bool {
		return "boolean"
	}
	// 只改写显式写了 type 标签的字段；无标签时交给驱动按 Go 类型推断（自增主键要生成 serial）
	tag, ok := field.TagSettings["TYPE"]
	if !ok {
		return d.Dialector.DataTypeOf(field)
	}
	// ops_instance_group 这类关联表把 ID 写成 varchar(11) 而 Go 字段是 int：PG 侧按整数建列，
	// 否则与 bigint 主键 JOIN 会报 42883（operator does not exist: bigint = character varying）
	if field.GORMDataType == schema.Int || field.GORMDataType == schema.Uint {
		if base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "("); base == "varchar" || base == "char" {
			return "bigint"
		}
	}
	if pgType, matched := pgTypeOfTag(tag); matched {
		return pgType
	}
	return d.Dialector.DataTypeOf(field)
}

var pgIntTagBases = map[string]string{
	"int": "integer", "integer": "integer", "mediumint": "integer",
	"smallint": "smallint", "tinyint": "smallint",
	"bigint": "bigint",
}

// pgTypeOfTag 只处理确定不兼容的标签；其余交回驱动按原样输出。
func pgTypeOfTag(tag string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(tag))
	base := lower
	if i := strings.IndexByte(base, '('); i >= 0 {
		base = base[:i]
	} else {
		// 无宽度修饰的标签里，只有整型别名和 datetime 需要改写
		switch base {
		case "int", "integer", "mediumint", "smallint", "tinyint", "bigint":
			return pgIntTagBases[base], true
		case "datetime", "timestamp":
			return "timestamptz", true
		default:
			return "", false
		}
	}
	if mapped, ok := pgIntTagBases[base]; ok {
		return mapped, true
	}
	switch base {
	case "datetime", "timestamp":
		return "timestamptz", true
	case "longtext", "mediumtext":
		return "text", true
	}
	// varchar(n)/char(n)/text/decimal(p,s) 等在 PG 中同样合法
	return "", false
}
