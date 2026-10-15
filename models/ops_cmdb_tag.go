package models

// OpsCmdbTag 资产标签
type OpsCmdbTag struct {
	Name  string `gorm:"column:name;type:varchar(64);not null;uniqueIndex:uk_cmdb_tag_name;comment:标签名称" json:"name"`
	Color string `gorm:"column:color;type:varchar(32);comment:展示颜色" json:"color"`
	Base
}

const TableOpsCmdbTag = "ops_cmdb_tag"

func (OpsCmdbTag) TableName() string {
	return TableOpsCmdbTag
}

// OpsCmdbAssetTag 资产与标签关联表
type OpsCmdbAssetTag struct {
	AssetId int `gorm:"column:asset_id;type:int(11);not null;uniqueIndex:uk_cmdb_asset_tag;comment:资产ID" json:"assetId"`
	TagId   int `gorm:"column:tag_id;type:int(11);not null;uniqueIndex:uk_cmdb_asset_tag;comment:标签ID" json:"tagId"`
}

const TableOpsCmdbAssetTag = "ops_cmdb_asset_tag"

func (OpsCmdbAssetTag) TableName() string {
	return TableOpsCmdbAssetTag
}
