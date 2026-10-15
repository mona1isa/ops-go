package cmdb

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// TagService 资产标签管理
type TagService struct{}

// List 全量标签（数量有限，不分页）
func (s *TagService) List() ([]models.OpsCmdbTag, error) {
	var tags []models.OpsCmdbTag
	if err := models.DB.Where("del_flag = ?", "0").Order("id").Find(&tags).Error; err != nil {
		log.Println("查询标签失败:", err)
		return nil, errors.New("查询标签失败")
	}
	return tags, nil
}

// Add 新增标签
func (s *TagService) Add(tag *models.OpsCmdbTag) error {
	if strings.TrimSpace(tag.Name) == "" {
		return errors.New("标签名称不能为空")
	}
	var count int64
	if err := models.DB.Model(&models.OpsCmdbTag{}).
		Where("del_flag = ? AND name = ?", "0", tag.Name).Count(&count).Error; err != nil {
		log.Println("校验标签失败:", err)
		return errors.New("校验标签失败")
	}
	if count > 0 {
		return errors.New("标签名称已存在")
	}
	tag.DelFlag = "0"
	if err := models.DB.Create(tag).Error; err != nil {
		log.Println("新增标签失败:", err)
		return errors.New("新增标签失败")
	}
	return nil
}

// Edit 编辑标签
func (s *TagService) Edit(tag *models.OpsCmdbTag) error {
	var existing models.OpsCmdbTag
	if err := models.DB.Where("id = ? AND del_flag = ?", tag.ID, "0").First(&existing).Error; err != nil {
		return errors.New("标签不存在")
	}
	if strings.TrimSpace(tag.Name) == "" {
		return errors.New("标签名称不能为空")
	}
	var count int64
	err := models.DB.Model(&models.OpsCmdbTag{}).
		Where("del_flag = ? AND name = ? AND id != ?", "0", tag.Name, tag.ID).Count(&count).Error
	if err != nil {
		log.Println("校验标签失败:", err)
		return errors.New("校验标签失败")
	}
	if count > 0 {
		return errors.New("标签名称已存在")
	}
	updates := map[string]any{"name": tag.Name, "color": tag.Color, "update_by": tag.UpdateBy}
	if err := models.DB.Model(&existing).Updates(updates).Error; err != nil {
		log.Println("编辑标签失败:", err)
		return errors.New("编辑标签失败")
	}
	return nil
}

// Delete 删除标签并清理资产关联
func (s *TagService) Delete(id int) error {
	var existing models.OpsCmdbTag
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&existing).Error; err != nil {
		return errors.New("标签不存在")
	}
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&existing).Update("del_flag", "1").Error; err != nil {
			return err
		}
		return tx.Where("tag_id = ?", id).Delete(&models.OpsCmdbAssetTag{}).Error
	})
	if err != nil {
		log.Println("删除标签失败:", err)
		return errors.New("删除标签失败")
	}
	return nil
}

// SetAssetTags 全量替换资产标签并记录变更
func (s *TagService) SetAssetTags(assetId int, tagIds []int, userId string) error {
	var asset models.OpsCmdbAsset
	if err := models.DB.Where("id = ? AND del_flag = ?", assetId, "0").First(&asset).Error; err != nil {
		return errors.New("资产不存在")
	}
	if len(tagIds) > 0 {
		var valid int64
		if err := models.DB.Model(&models.OpsCmdbTag{}).
			Where("id IN ? AND del_flag = ?", tagIds, "0").Count(&valid).Error; err != nil {
			log.Println("校验标签失败:", err)
			return errors.New("校验标签失败")
		}
		if int(valid) != len(tagIds) {
			return errors.New("包含不存在或已删除的标签")
		}
	}

	var before []models.OpsCmdbTag
	if tagMap, err := tagsOfAssets([]int{assetId}); err != nil {
		return err
	} else {
		before = tagMap[assetId]
	}

	err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("asset_id = ?", assetId).Delete(&models.OpsCmdbAssetTag{}).Error; err != nil {
			return err
		}
		for _, tid := range tagIds {
			link := models.OpsCmdbAssetTag{AssetId: assetId, TagId: tid}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Println("更新资产标签失败:", err)
		return errors.New("更新资产标签失败")
	}

	names := func(tags []models.OpsCmdbTag) string {
		if len(tags) == 0 {
			return "无"
		}
		parts := make([]string, 0, len(tags))
		for _, t := range tags {
			parts = append(parts, t.Name)
		}
		return strings.Join(parts, "、")
	}
	var after []models.OpsCmdbTag
	if len(tagIds) > 0 {
		if err := models.DB.Where("id IN ?", tagIds).Find(&after).Error; err != nil {
			log.Println("查询变更后标签失败:", err)
		}
	}
	writeChangeLog(assetId, "tag", fmt.Sprintf("标签变更：%s → %s", names(before), names(after)), userId)
	return nil
}
