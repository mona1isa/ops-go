package models

// OpsCmdbChangeLog 资产变更记录
type OpsCmdbChangeLog struct {
	AssetId int    `gorm:"column:asset_id;type:int(11);not null;index:idx_cmdb_changelog_asset;comment:资产ID" json:"assetId"`
	Action  string `gorm:"column:action;type:varchar(32);not null;comment:动作（create/update/status/import/relation/tag/delete）" json:"action"`
	Detail  string `gorm:"column:detail;type:varchar(1000);comment:变更摘要" json:"detail"`
	Base
}

const TableOpsCmdbChangeLog = "ops_cmdb_change_log"

func (OpsCmdbChangeLog) TableName() string {
	return TableOpsCmdbChangeLog
}
