package instance

import (
	"errors"
	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
	"log"
)

type ScriptService struct{}

func (s *ScriptService) List(pageNum, pageSize int, name string) (models.PageResult[models.OpsScript], error) {
	return models.Paginate[models.OpsScript](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		if name != "" {
			db = db.Where("name like ?", "%"+name+"%")
		}
		return db
	})
}

func (s *ScriptService) GetByID(id int) (*models.OpsScript, error) {
	var script models.OpsScript
	if err := models.DB.Where("id = ? AND del_flag = ?", id, 0).First(&script).Error; err != nil {
		return nil, errors.New("脚本不存在")
	}
	return &script, nil
}

func (s *ScriptService) Add(script *models.OpsScript) error {
	if script.Name == "" {
		return errors.New("脚本名称不能为空")
	}
	if script.Type == "" {
		script.Type = "bash"
	}
	if err := models.DB.Create(script).Error; err != nil {
		log.Printf("新增脚本失败: %v", err)
		return errors.New("新增失败")
	}
	return nil
}

func (s *ScriptService) Edit(script *models.OpsScript) error {
	if script.ID == 0 {
		return errors.New("脚本ID不能为空")
	}
	var existing models.OpsScript
	if err := models.DB.Where("id = ? AND del_flag = ?", script.ID, 0).First(&existing).Error; err != nil {
		return errors.New("脚本不存在")
	}
	updates := map[string]interface{}{
		"name":      script.Name,
		"content":   script.Content,
		"type":      script.Type,
		"remark":    script.Remark,
		"update_by": script.UpdateBy,
	}
	if err := models.DB.Model(&existing).Updates(updates).Error; err != nil {
		log.Printf("编辑脚本失败: %v", err)
		return errors.New("编辑失败")
	}
	return nil
}

func (s *ScriptService) Delete(id int) error {
	var existing models.OpsScript
	if err := models.DB.Where("id = ? AND del_flag = ?", id, 0).First(&existing).Error; err != nil {
		return errors.New("脚本不存在")
	}
	if err := models.DB.Model(&existing).Update("del_flag", 1).Error; err != nil {
		return errors.New("删除失败")
	}
	return nil
}
