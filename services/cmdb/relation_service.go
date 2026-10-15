package cmdb

import (
	"errors"
	"log"

	"github.com/zhany/ops-go/models"
)

// RelationService 资产关联关系与拓扑
type RelationService struct{}

// RelationRow 关系列表行：带两端资产名称
type RelationRow struct {
	Id           int    `json:"id"`
	SourceId     int    `json:"sourceId"`
	SourceName   string `json:"sourceName"`
	TargetId     int    `json:"targetId"`
	TargetName   string `json:"targetName"`
	RelationType string `json:"relationType"`
}

// GraphNode 拓扑节点
type GraphNode struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	ModelType string `json:"modelType"`
	Status    int8   `json:"status"`
}

// GraphEdge 拓扑边
type GraphEdge struct {
	Source int    `json:"source"`
	Target int    `json:"target"`
	Type   string `json:"type"`
}

// GraphData 拓扑图数据
type GraphData struct {
	Nodes     []GraphNode `json:"nodes"`
	Edges     []GraphEdge `json:"edges"`
	Truncated bool        `json:"truncated"`
}

func validRelationType(t string) bool {
	for _, v := range models.CmdbRelationTypes {
		if t == v {
			return true
		}
	}
	return false
}

// Add 新增关联关系
func (s *RelationService) Add(sourceId, targetId int, relationType, userId string) error {
	if sourceId == targetId {
		return errors.New("源资产与目标资产不能相同")
	}
	if !validRelationType(relationType) {
		return errors.New("关系类型不合法")
	}
	var count int64
	if err := models.DB.Model(&models.OpsCmdbAsset{}).
		Where("id IN ? AND del_flag = ?", []int{sourceId, targetId}, "0").Count(&count).Error; err != nil {
		log.Println("校验资产失败:", err)
		return errors.New("校验资产失败")
	}
	if count != 2 {
		return errors.New("资产不存在或已删除")
	}
	dup := int64(0)
	err := models.DB.Model(&models.OpsCmdbRelation{}).
		Where("source_id = ? AND target_id = ? AND relation_type = ?", sourceId, targetId, relationType).
		Count(&dup).Error
	if err != nil {
		log.Println("校验关联关系失败:", err)
		return errors.New("校验关联关系失败")
	}
	if dup > 0 {
		return errors.New("相同关联关系已存在")
	}
	rel := models.OpsCmdbRelation{
		SourceId:     sourceId,
		TargetId:     targetId,
		RelationType: relationType,
		Base:         models.Base{CreateBy: userId, UpdateBy: userId},
	}
	if err := models.DB.Create(&rel).Error; err != nil {
		log.Println("新增关联关系失败:", err)
		return errors.New("新增关联关系失败")
	}
	writeChangeLog(sourceId, "relation", "新增关联", userId)
	return nil
}

// Delete 删除关联关系
func (s *RelationService) Delete(id int, userId string) error {
	var rel models.OpsCmdbRelation
	if err := models.DB.Where("id = ?", id).First(&rel).Error; err != nil {
		return errors.New("关联关系不存在")
	}
	if err := models.DB.Delete(&rel).Error; err != nil {
		log.Println("删除关联关系失败:", err)
		return errors.New("删除关联关系失败")
	}
	writeChangeLog(rel.SourceId, "relation", "删除关联", userId)
	return nil
}

// assetBriefMap 批量取资产简要信息
func assetBriefMap(ids []int) map[int]models.OpsCmdbAsset {
	out := make(map[int]models.OpsCmdbAsset)
	if len(ids) == 0 {
		return out
	}
	var assets []models.OpsCmdbAsset
	if err := models.DB.Where("id IN ? AND del_flag = ?", ids, "0").Find(&assets).Error; err != nil {
		log.Println("查询资产失败:", err)
		return out
	}
	for _, a := range assets {
		out[a.ID] = a
	}
	return out
}

// ListByAsset 查询某资产的全部关联（作为源或目标）
func (s *RelationService) ListByAsset(assetId int) ([]RelationRow, error) {
	var rels []models.OpsCmdbRelation
	if err := models.DB.Where("source_id = ? OR target_id = ?", assetId, assetId).Find(&rels).Error; err != nil {
		log.Println("查询关联关系失败:", err)
		return nil, errors.New("查询关联关系失败")
	}
	ids := make([]int, 0, len(rels)*2)
	for _, r := range rels {
		ids = append(ids, r.SourceId, r.TargetId)
	}
	assets := assetBriefMap(ids)
	rows := make([]RelationRow, 0, len(rels))
	for _, r := range rels {
		src, dst := assets[r.SourceId], assets[r.TargetId]
		if src.ID == 0 || dst.ID == 0 {
			continue // 一端资产已删除
		}
		rows = append(rows, RelationRow{
			Id: r.ID, SourceId: r.SourceId, SourceName: src.Name,
			TargetId: r.TargetId, TargetName: dst.Name, RelationType: r.RelationType,
		})
	}
	return rows, nil
}

const graphNodeLimit = 300

// allActiveAssets 全量未删除资产
func allActiveAssets() ([]models.OpsCmdbAsset, error) {
	var assets []models.OpsCmdbAsset
	if err := models.DB.Where("del_flag = ?", "0").Find(&assets).Error; err != nil {
		log.Println("查询资产失败:", err)
		return nil, errors.New("查询资产失败")
	}
	return assets, nil
}

// loadActiveEdges 加载两端均存在的关联
func loadActiveEdges(alive map[int]bool) ([]models.OpsCmdbRelation, error) {
	var rels []models.OpsCmdbRelation
	if err := models.DB.Find(&rels).Error; err != nil {
		log.Println("查询关联关系失败:", err)
		return nil, errors.New("查询关联关系失败")
	}
	out := make([]models.OpsCmdbRelation, 0, len(rels))
	for _, r := range rels {
		if alive[r.SourceId] && alive[r.TargetId] {
			out = append(out, r)
		}
	}
	return out, nil
}

// GraphAll 全量拓扑（可按模型类型筛选，节点数超限截断）
func (s *RelationService) GraphAll(modelTypes []string) (*GraphData, error) {
	assets, err := allActiveAssets()
	if err != nil {
		return nil, err
	}
	typeFilter := make(map[string]bool)
	for _, t := range modelTypes {
		typeFilter[t] = true
	}
	alive := make(map[int]bool)
	nodes := make([]GraphNode, 0, len(assets))
	for _, a := range assets {
		if len(typeFilter) > 0 && !typeFilter[a.ModelType] {
			continue
		}
		if len(nodes) >= graphNodeLimit {
			break
		}
		alive[a.ID] = true
		nodes = append(nodes, GraphNode{Id: a.ID, Name: a.Name, ModelType: a.ModelType, Status: a.Status})
	}
	rels, err := loadActiveEdges(alive)
	if err != nil {
		return nil, err
	}
	edges := make([]GraphEdge, 0, len(rels))
	for _, r := range rels {
		edges = append(edges, GraphEdge{Source: r.SourceId, Target: r.TargetId, Type: r.RelationType})
	}
	return &GraphData{Nodes: nodes, Edges: edges, Truncated: len(nodes) >= graphNodeLimit}, nil
}

// Graph 以某资产为中心的邻域拓扑（BFS，depth 层，限制 1-3）
func (s *RelationService) Graph(rootId, depth int) (*GraphData, error) {
	if depth < 1 {
		depth = 1
	}
	if depth > 3 {
		depth = 3
	}
	assets, err := allActiveAssets()
	if err != nil {
		return nil, err
	}
	byId := make(map[int]models.OpsCmdbAsset)
	for _, a := range assets {
		byId[a.ID] = a
	}
	if _, ok := byId[rootId]; !ok {
		return nil, errors.New("资产不存在")
	}
	var allRels []models.OpsCmdbRelation
	if err := models.DB.Find(&allRels).Error; err != nil {
		log.Println("查询关联关系失败:", err)
		return nil, errors.New("查询关联关系失败")
	}

	visited := map[int]bool{rootId: true}
	frontier := []int{rootId}
	for d := 0; d < depth; d++ {
		next := make([]int, 0)
		for _, id := range frontier {
			for _, r := range allRels {
				other := 0
				if r.SourceId == id {
					other = r.TargetId
				} else if r.TargetId == id {
					other = r.SourceId
				} else {
					continue
				}
				if _, ok := byId[other]; !ok || visited[other] {
					continue
				}
				visited[other] = true
				next = append(next, other)
			}
		}
		frontier = next
	}

	nodes := make([]GraphNode, 0, len(visited))
	for id := range visited {
		a := byId[id]
		nodes = append(nodes, GraphNode{Id: a.ID, Name: a.Name, ModelType: a.ModelType, Status: a.Status})
	}
	edges := make([]GraphEdge, 0)
	for _, r := range allRels {
		if visited[r.SourceId] && visited[r.TargetId] {
			edges = append(edges, GraphEdge{Source: r.SourceId, Target: r.TargetId, Type: r.RelationType})
		}
	}
	return &GraphData{Nodes: nodes, Edges: edges, Truncated: false}, nil
}

// AssetOptions 资产下拉选项（关联目标选择器用）
type AssetOption struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Ip   string `json:"ip"`
}

func (s *AssetService) AssetOptions(keyword string) ([]AssetOption, error) {
	db := models.DB.Model(&models.OpsCmdbAsset{}).Where("del_flag = ?", "0")
	if keyword != "" {
		kw := "%" + escapeLike(keyword) + "%"
		db = db.Where("name LIKE ? OR ip LIKE ?", kw, kw)
	}
	var assets []models.OpsCmdbAsset
	if err := db.Limit(50).Order("id DESC").Find(&assets).Error; err != nil {
		log.Println("查询资产选项失败:", err)
		return nil, errors.New("查询资产列表失败")
	}
	options := make([]AssetOption, 0, len(assets))
	for _, a := range assets {
		options = append(options, AssetOption{Id: a.ID, Name: a.Name, Ip: a.Ip})
	}
	return options, nil
}
