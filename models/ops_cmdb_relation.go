package models

// 资产关联类型
const (
	CmdbRelRunsOn   = "runs_on"     // 部署于
	CmdbRelBindsTo  = "binds_to"    // 绑定
	CmdbRelConnects = "connects_to" // 连接
	CmdbRelDepends  = "depends"     // 依赖
	CmdbRelResolves = "resolves"    // 解析
)

// CmdbRelationTypes 全部合法关联类型
var CmdbRelationTypes = []string{
	CmdbRelRunsOn, CmdbRelBindsTo, CmdbRelConnects, CmdbRelDepends, CmdbRelResolves,
}

// OpsCmdbRelation 资产关联关系
type OpsCmdbRelation struct {
	SourceId     int    `gorm:"column:source_id;type:int(11);not null;uniqueIndex:uk_cmdb_relation;index:idx_cmdb_relation_source;comment:源资产ID" json:"sourceId"`
	TargetId     int    `gorm:"column:target_id;type:int(11);not null;uniqueIndex:uk_cmdb_relation;index:idx_cmdb_relation_target;comment:目标资产ID" json:"targetId"`
	RelationType string `gorm:"column:relation_type;type:varchar(32);not null;uniqueIndex:uk_cmdb_relation;comment:关系类型" json:"relationType"`
	Base
}

const TableOpsCmdbRelation = "ops_cmdb_relation"

func (OpsCmdbRelation) TableName() string {
	return TableOpsCmdbRelation
}
