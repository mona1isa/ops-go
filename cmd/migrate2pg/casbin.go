package main

import (
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// newCasbinAdapter 建 casbin_rule 表，与运行时 models/casbin_handler.go 走同一套 adapter
func newCasbinAdapter(db *gorm.DB) (*gormadapter.Adapter, error) {
	return gormadapter.NewAdapterByDB(db)
}
