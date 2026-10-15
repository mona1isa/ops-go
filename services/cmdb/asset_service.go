package cmdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// AssetService 资产台账管理服务
type AssetService struct{}

// AssetFilter 资产列表查询条件
type AssetFilter struct {
	ModelType    string
	Status       *int8
	TagId        int
	Keyword      string
	ExpireWithin int // 距今 N 天内到期（0 不启用）
}

// AssetRow 资产列表行：资产字段 + 标签
type AssetRow struct {
	models.OpsCmdbAsset
	Tags []models.OpsCmdbTag `json:"tags"`
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// List 分页查询资产
func (s *AssetService) List(pageNum, pageSize int, filter AssetFilter) (models.PageResult[AssetRow], error) {
	result := models.PageResult[AssetRow]{}
	scope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("del_flag = ?", "0")
		if filter.ModelType != "" {
			db = db.Where("model_type = ?", filter.ModelType)
		}
		if filter.Status != nil {
			db = db.Where("status = ?", *filter.Status)
		}
		if filter.Keyword != "" {
			kw := "%" + escapeLike(filter.Keyword) + "%"
			db = db.Where("name LIKE ? OR ip LIKE ? OR sn LIKE ?", kw, kw, kw)
		}
		if filter.TagId > 0 {
			db = db.Where("id IN (?)",
				models.DB.Table(models.TableOpsCmdbAssetTag).Select("asset_id").Where("tag_id = ?", filter.TagId))
		}
		if filter.ExpireWithin > 0 {
			db = db.Where("expired_at IS NOT NULL AND expired_at <= DATE_ADD(NOW(), INTERVAL ? DAY) AND expired_at >= NOW()", filter.ExpireWithin)
		}
		return db.Order("id DESC")
	}
	page, err := models.Paginate[models.OpsCmdbAsset](models.DB, pageNum, pageSize, scope)
	if err != nil {
		log.Println("查询资产列表失败:", err)
		return result, errors.New("查询资产列表失败")
	}

	rows := make([]AssetRow, 0, len(page.Data))
	ids := make([]int, 0, len(page.Data))
	for _, a := range page.Data {
		ids = append(ids, a.ID)
		rows = append(rows, AssetRow{OpsCmdbAsset: a, Tags: []models.OpsCmdbTag{}})
	}
	if len(ids) > 0 {
		tagMap, err := tagsOfAssets(ids)
		if err != nil {
			return result, err
		}
		for i := range rows {
			if tags, ok := tagMap[rows[i].ID]; ok {
				rows[i].Tags = tags
			}
		}
	}
	result = models.PageResult[AssetRow]{
		Total:     page.Total,
		PageNum:   page.PageNum,
		PageSize:  page.PageSize,
		TotalPage: page.TotalPage,
		Data:      rows,
	}
	return result, nil
}

// tagsOfAssets 批量查询资产标签，返回 assetId -> tags
func tagsOfAssets(assetIds []int) (map[int][]models.OpsCmdbTag, error) {
	out := make(map[int][]models.OpsCmdbTag)
	var links []models.OpsCmdbAssetTag
	if err := models.DB.Where("asset_id IN ?", assetIds).Find(&links).Error; err != nil {
		log.Println("查询资产标签关联失败:", err)
		return nil, errors.New("查询资产标签失败")
	}
	if len(links) == 0 {
		return out, nil
	}
	tagIds := make([]int, 0, len(links))
	for _, l := range links {
		tagIds = append(tagIds, l.TagId)
	}
	var tags []models.OpsCmdbTag
	if err := models.DB.Where("id IN ? AND del_flag = ?", tagIds, "0").Find(&tags).Error; err != nil {
		log.Println("查询标签失败:", err)
		return nil, errors.New("查询标签失败")
	}
	tagById := make(map[int]models.OpsCmdbTag)
	for _, t := range tags {
		tagById[t.ID] = t
	}
	for _, l := range links {
		if t, ok := tagById[l.TagId]; ok {
			out[l.AssetId] = append(out[l.AssetId], t)
		}
	}
	return out, nil
}

// validateAsset 校验模型类型、名称与属性 JSON
func validateAsset(asset *models.OpsCmdbAsset) error {
	if strings.TrimSpace(asset.Name) == "" {
		return errors.New("资产名称不能为空")
	}
	known := false
	for _, t := range models.CmdbModelTypes {
		if asset.ModelType == t {
			known = true
			break
		}
	}
	if !known {
		return errors.New("资产模型类型不合法")
	}
	if asset.Status < models.CmdbStatusPurchased || asset.Status > models.CmdbStatusRetired {
		asset.Status = models.CmdbStatusRunning
	}
	if asset.Attrs != "" && !json.Valid([]byte(asset.Attrs)) {
		return errors.New("资产属性不是合法的 JSON")
	}
	return nil
}

func (s *AssetService) checkNameUnique(modelType, name string, excludeId int) error {
	var count int64
	err := models.DB.Model(&models.OpsCmdbAsset{}).
		Where("del_flag = ? AND model_type = ? AND name = ? AND id != ?", "0", modelType, name, excludeId).
		Count(&count).Error
	if err != nil {
		log.Println("校验资产名称失败:", err)
		return errors.New("校验资产名称失败")
	}
	if count > 0 {
		return errors.New("同类型下资产名称已存在")
	}
	return nil
}

// Add 新增资产
func (s *AssetService) Add(asset *models.OpsCmdbAsset) error {
	if err := validateAsset(asset); err != nil {
		return err
	}
	if err := s.checkNameUnique(asset.ModelType, asset.Name, 0); err != nil {
		return err
	}
	asset.Source = "manual"
	asset.DelFlag = "0"
	if err := models.DB.Create(asset).Error; err != nil {
		log.Println("新增资产失败:", err)
		return errors.New("新增资产失败")
	}
	writeChangeLog(asset.ID, "create", fmt.Sprintf("创建资产：%s", asset.Name), asset.CreateBy)
	return nil
}

// Edit 编辑资产（模型类型创建后不可变更）
func (s *AssetService) Edit(asset *models.OpsCmdbAsset) error {
	var existing models.OpsCmdbAsset
	if err := models.DB.Where("id = ? AND del_flag = ?", asset.ID, "0").First(&existing).Error; err != nil {
		return errors.New("资产不存在")
	}
	asset.ModelType = existing.ModelType
	if err := validateAsset(asset); err != nil {
		return err
	}
	if err := s.checkNameUnique(existing.ModelType, asset.Name, asset.ID); err != nil {
		return err
	}
	updates := map[string]any{
		"name":       asset.Name,
		"sn":         asset.Sn,
		"ip":         asset.Ip,
		"owner":      asset.Owner,
		"provider":   asset.Provider,
		"location":   asset.Location,
		"expired_at": asset.ExpiredAt,
		"attrs":      asset.Attrs,
		"remark":     asset.Remark,
		"update_by":  asset.UpdateBy,
	}
	if err := models.DB.Model(&existing).Updates(updates).Error; err != nil {
		log.Println("编辑资产失败:", err)
		return errors.New("编辑资产失败")
	}
	writeChangeLog(asset.ID, "update", fmt.Sprintf("编辑资产：%s", asset.Name), asset.UpdateBy)
	return nil
}

// Detail 查询资产详情
func (s *AssetService) Detail(id int) (*AssetRow, error) {
	var asset models.OpsCmdbAsset
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&asset).Error; err != nil {
		return nil, errors.New("资产不存在")
	}
	row := &AssetRow{OpsCmdbAsset: asset, Tags: []models.OpsCmdbTag{}}
	tagMap, err := tagsOfAssets([]int{id})
	if err != nil {
		return nil, err
	}
	if tags, ok := tagMap[id]; ok {
		row.Tags = tags
	}
	return row, nil
}

// Delete 软删除资产并清理标签、关系
func (s *AssetService) Delete(id int, userId string) error {
	var asset models.OpsCmdbAsset
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&asset).Error; err != nil {
		return errors.New("资产不存在")
	}
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&asset).Update("del_flag", "1").Error; err != nil {
			return err
		}
		if err := tx.Where("asset_id = ?", id).Delete(&models.OpsCmdbAssetTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("source_id = ? OR target_id = ?", id, id).Delete(&models.OpsCmdbRelation{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Println("删除资产失败:", err)
		return errors.New("删除资产失败")
	}
	writeChangeLog(id, "delete", fmt.Sprintf("删除资产：%s", asset.Name), userId)
	return nil
}

// Transition 生命周期状态流转
func (s *AssetService) Transition(id int, status int8, remark, userId string) error {
	var asset models.OpsCmdbAsset
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&asset).Error; err != nil {
		return errors.New("资产不存在")
	}
	if status < models.CmdbStatusPurchased || status > models.CmdbStatusRetired {
		return errors.New("生命周期状态不合法")
	}
	if status == asset.Status {
		return errors.New("目标状态与当前状态相同")
	}
	from := statusText(asset.Status)
	to := statusText(status)
	updates := map[string]any{"status": status, "update_by": userId}
	if remark != "" {
		updates["remark"] = remark
	}
	if err := models.DB.Model(&asset).Updates(updates).Error; err != nil {
		log.Println("资产状态流转失败:", err)
		return errors.New("状态流转失败")
	}
	detail := fmt.Sprintf("状态流转：%s → %s", from, to)
	if remark != "" {
		detail += fmt.Sprintf("（%s）", remark)
	}
	writeChangeLog(id, "status", detail, userId)
	return nil
}

// ImportFromInstances 从已纳管主机批量导入服务器资产，跳过同名/同IP
func (s *AssetService) ImportFromInstances(userId string) (int, error) {
	var instances []models.OpsInstance
	if err := models.DB.Where("del_flag = ?", "0").Find(&instances).Error; err != nil {
		log.Println("查询主机失败:", err)
		return 0, errors.New("查询主机列表失败")
	}
	created := 0
	for _, inst := range instances {
		var count int64
		err := models.DB.Model(&models.OpsCmdbAsset{}).
			Where("del_flag = ? AND model_type = ? AND (name = ? OR (ip <> '' AND ip = ?))", "0", models.CmdbModelServer, inst.Name, inst.Ip).
			Count(&count).Error
		if err != nil {
			log.Printf("导入资产查重失败 instance=%d: %v", inst.ID, err)
			continue
		}
		if count > 0 {
			continue
		}
		attrs, _ := json.Marshal(map[string]any{
			"cpu": inst.Cpu, "memGb": inst.MemMb, "diskGb": inst.DiskGb, "os": inst.Os,
		})
		asset := models.OpsCmdbAsset{
			ModelType:  models.CmdbModelServer,
			Name:       inst.Name,
			Ip:         inst.Ip,
			Status:     models.CmdbStatusRunning,
			Attrs:      string(attrs),
			Source:     "import",
			InstanceId: inst.ID,
			Base:       models.Base{CreateBy: userId, UpdateBy: userId},
		}
		if err := models.DB.Create(&asset).Error; err != nil {
			log.Printf("导入资产失败 instance=%d: %v", inst.ID, err)
			continue
		}
		writeChangeLog(asset.ID, "import", fmt.Sprintf("从纳管主机导入：%s(%s)", inst.Name, inst.Ip), userId)
		created++
	}
	return created, nil
}

func statusText(status int8) string {
	switch status {
	case models.CmdbStatusPurchased:
		return "采购"
	case models.CmdbStatusRacked:
		return "上架"
	case models.CmdbStatusRunning:
		return "运行"
	case models.CmdbStatusMaintain:
		return "维护"
	case models.CmdbStatusRetired:
		return "退役"
	default:
		return fmt.Sprintf("未知(%d)", status)
	}
}
