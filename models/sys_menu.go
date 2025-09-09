package models

type SysMenu struct {
	Name      string `gorm:"type:varchar(64);not null;unique;comment:菜单名称" json:"name"`
	ParentId  int    `gorm:"type:int(11);comment:父菜单ID" json:"parentId"`
	OrderNum  int    `gorm:"type:int(4);comment:显示顺序" json:"orderNum"`
	Path      string `gorm:"type:varchar(200);comment:路由地址" json:"path"`
	Component string `gorm:"type:varchar(255);comment:组件路径" json:"component"`
	IsAffix   bool   `gorm:"type:tinyint(1);comment:是否固定（1 是 0 否）" json:"isAffix"`
	IsFrame   int    `gorm:"type:tinyint(1);comment:是否为外链（1是 0否）" json:"isFrame"`
	IsCache   int    `gorm:"type:tinyint(1);comment:是否缓存（1缓存 0不缓存）" json:"isCache"`
	Type      string `gorm:"type:char(1);comment:菜单类型（M目录 C菜单 F按钮）" json:"type"`
	Visible   string `gorm:"type:char(1);default:0;comment:菜单状态（1显示 0隐藏）" json:"visible"`
	Status    string `gorm:"type:char(1);default:0;comment:菜单状态（1正常 0停用）" json:"status"`
	Perms     string `gorm:"type:varchar(100);comment:权限标识" json:"perms"`
	Icon      string `gorm:"type:varchar(100);comment:菜单图标" json:"icon"`
	Base
}

const TableSysMenu = "sys_menu"

func (SysMenu) TableName() string {
	return TableSysMenu
}
