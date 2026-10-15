package models

import "time"

// 资产模型类型
const (
	CmdbModelServer     = "server"     // 服务器
	CmdbModelNetwork    = "network"    // 网络设备
	CmdbModelDatabase   = "database"   // 数据库
	CmdbModelMiddleware = "middleware" // 中间件
	CmdbModelDomain     = "domain"     // 域名
	CmdbModelSsl        = "ssl"        // SSL证书
)

// CmdbModelTypes 全部合法模型类型
var CmdbModelTypes = []string{
	CmdbModelServer, CmdbModelNetwork, CmdbModelDatabase,
	CmdbModelMiddleware, CmdbModelDomain, CmdbModelSsl,
}

// 资产生命周期状态
const (
	CmdbStatusPurchased = 1 // 采购
	CmdbStatusRacked    = 2 // 上架
	CmdbStatusRunning   = 3 // 运行
	CmdbStatusMaintain  = 4 // 维护
	CmdbStatusRetired   = 5 // 退役
)

// OpsCmdbAsset 资产主表
type OpsCmdbAsset struct {
	ModelType  string     `gorm:"column:model_type;type:varchar(32);not null;uniqueIndex:uk_cmdb_asset_model_name;comment:资产模型类型" json:"modelType"`
	Name       string     `gorm:"column:name;type:varchar(128);not null;uniqueIndex:uk_cmdb_asset_model_name;comment:资产名称" json:"name"`
	Sn         string     `gorm:"column:sn;type:varchar(128);comment:资产编号" json:"sn"`
	Ip         string     `gorm:"column:ip;type:varchar(64);index:idx_cmdb_asset_ip;comment:IP地址（域名/证书可为空）" json:"ip"`
	Status     int8       `gorm:"column:status;type:tinyint;default:1;comment:生命周期（1采购 2上架 3运行 4维护 5退役）" json:"status"`
	Owner      string     `gorm:"column:owner;type:varchar(64);comment:负责人" json:"owner"`
	Provider   string     `gorm:"column:provider;type:varchar(128);comment:供应商/云厂商" json:"provider"`
	Location   string     `gorm:"column:location;type:varchar(128);comment:机房/区域" json:"location"`
	ExpiredAt  *time.Time `gorm:"column:expired_at;comment:到期时间（域名/证书）" json:"expiredAt"`
	Attrs      string     `gorm:"column:attrs;type:text;comment:模型专属属性（JSON字符串）" json:"attrs"`
	Source     string     `gorm:"column:source;type:varchar(32);default:manual;comment:来源（manual 手动 import 主机导入）" json:"source"`
	InstanceId int        `gorm:"column:instance_id;type:int(11);default:0;comment:来源主机ID（导入资产，0 表示无）" json:"instanceId"`
	Base
}

const TableOpsCmdbAsset = "ops_cmdb_asset"

func (OpsCmdbAsset) TableName() string {
	return TableOpsCmdbAsset
}
