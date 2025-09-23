package models

type SysMenu struct {
	Name          string `gorm:"type:varchar(64);not null;unique;comment:菜单名称" json:"name"`
	ParentId      int    `gorm:"type:int(11);comment:父菜单ID" json:"parentId"`
	OrderNum      int    `gorm:"type:int(4);comment:显示顺序" json:"orderNum"`
	Path          string `gorm:"type:varchar(200);comment:路由地址" json:"path"`
	Component     string `gorm:"type:varchar(255);comment:组件路径" json:"component"`
	IsAffix       bool   `gorm:"type:tinyint(1);comment:是否固定（1 是 0 否）" json:"isAffix"`
	IsIframe      bool   `gorm:"type:tinyint(1);comment:是否为内链（1是 0否）" json:"isIframe"`
	IsLink        bool   `gorm:"type:tinyint(1);comment:是否为外链（1是 0否）" json:"isLink"`
	KeepAlive     bool   `gorm:"type:tinyint(1);comment:是否缓存（true缓存 false不缓存）" json:"keepAlive"`
	Type          string `gorm:"type:char(1);comment:菜单类型（M目录 C菜单 F按钮）" json:"type"`
	IsHide        bool   `gorm:"type:tinyint(1);comment:是否隐藏（1隐藏 0显示）" json:"isHide"`
	Status        bool   `gorm:"type:char(1);default:0;comment:菜单状态（1正常 0停用）" json:"status"`
	Url           string `gorm:"type:varchar(128);default:null;comment:外链地址" json:"url"`
	Perms         string `gorm:"type:varchar(100);comment:权限标识" json:"perms"`
	Icon          string `gorm:"type:varchar(100);comment:菜单图标" json:"icon"`
	RequestUrl    string `gorm:"type:varchar(100);comment:接口请求地址" json:"requestUrl"`
	RequestMethod string `gorm:"type:varchar(10);comment:接口请求方法" json:"requestMethod"`
	Base
}

const TableSysMenu = "sys_menu"

func (SysMenu) TableName() string {
	return TableSysMenu
}
