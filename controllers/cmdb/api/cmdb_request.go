package api

import "time"

// 资产管理模块请求参数。所有 id 通过请求体传递，保证路由为静态路径以适配 Casbin 精确匹配。

// MonitorLikePageRequest 通用分页参数
type CmdbPageRequest struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

// AssetPageRequest 资产分页查询
type AssetPageRequest struct {
	CmdbPageRequest
	ModelType    string `json:"modelType"`
	Status       *int8  `json:"status"`
	TagId        int    `json:"tagId"`
	Keyword      string `json:"keyword"`
	ExpireWithin int    `json:"expireWithin"`
}

// AssetRequest 资产新增/编辑
type AssetRequest struct {
	Id        int        `json:"id"`
	ModelType string     `json:"modelType" binding:"required"`
	Name      string     `json:"name" binding:"required"`
	Sn        string     `json:"sn"`
	Ip        string     `json:"ip"`
	Status    int8       `json:"status"`
	Owner     string     `json:"owner"`
	Provider  string     `json:"provider"`
	Location  string     `json:"location"`
	ExpiredAt *time.Time `json:"expiredAt"`
	Attrs     string     `json:"attrs"`
	Remark    string     `json:"remark"`
}

// AssetIdRequest 资产详情/删除
type AssetIdRequest struct {
	Id int `json:"id" binding:"required"`
}

// AssetTransitionRequest 资产生命周期流转
type AssetTransitionRequest struct {
	Id     int    `json:"id" binding:"required"`
	Status int8   `json:"status" binding:"required"`
	Remark string `json:"remark"`
}

// AssetOptionsRequest 资产下拉选项
type AssetOptionsRequest struct {
	Keyword string `json:"keyword"`
}

// ChangeLogRequest 资产变更记录查询
type ChangeLogRequest struct {
	AssetId int `json:"assetId" binding:"required"`
	CmdbPageRequest
}

// TagRequest 标签新增/编辑
type TagRequest struct {
	Id    int    `json:"id"`
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
}

// TagIdRequest 标签删除
type TagIdRequest struct {
	Id int `json:"id" binding:"required"`
}

// TagSetRequest 全量设置资产标签
type TagSetRequest struct {
	AssetId int   `json:"assetId" binding:"required"`
	TagIds  []int `json:"tagIds"`
}

// RelationAddRequest 新增关联
type RelationAddRequest struct {
	SourceId     int    `json:"sourceId" binding:"required"`
	TargetId     int    `json:"targetId" binding:"required"`
	RelationType string `json:"relationType" binding:"required"`
}

// RelationIdRequest 删除关联
type RelationIdRequest struct {
	Id int `json:"id" binding:"required"`
}

// RelationListRequest 某资产的关联列表
type RelationListRequest struct {
	AssetId int `json:"assetId" binding:"required"`
}

// GraphAllRequest 全量拓扑（可按模型类型筛选）
type GraphAllRequest struct {
	ModelTypes []string `json:"modelTypes"`
}

// GraphRequest 邻域拓扑
type GraphRequest struct {
	RootId int `json:"rootId" binding:"required"`
	Depth  int `json:"depth"`
}

// SubnetPageRequest 网段分页查询
type SubnetPageRequest struct {
	CmdbPageRequest
	Keyword string `json:"keyword"`
}

// SubnetRequest 网段新增/编辑
type SubnetRequest struct {
	Id      int    `json:"id"`
	Name    string `json:"name" binding:"required"`
	Cidr    string `json:"cidr" binding:"required"`
	Gateway string `json:"gateway"`
	Vlan    string `json:"vlan"`
	Remark  string `json:"remark"`
}

// SubnetIdRequest 网段删除/使用查询
type SubnetIdRequest struct {
	Id int `json:"id" binding:"required"`
}
