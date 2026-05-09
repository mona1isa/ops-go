package instance

import (
	"errors"
	"log"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// TaskPipelineService 任务编排服务
type TaskPipelineService struct{}

// List 分页查询编排
func (s *TaskPipelineService) List(pageNum, pageSize int, name string) (models.PageResult[models.OpsTaskPipeline], error) {
	result, err := models.Paginate[models.OpsTaskPipeline](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		if name != "" {
			db = db.Where("name like ?", "%"+name+"%")
		}
		return db.Where("del_flag = ?", 0).Order("id desc")
	})
	if err != nil {
		return result, err
	}
	// 批量加载步骤数据
	if len(result.Data) > 0 {
		ids := make([]int, len(result.Data))
		for i, p := range result.Data {
			ids[i] = p.ID
		}
		var steps []models.OpsPipelineStep
		models.DB.Where("pipeline_id IN ? AND del_flag = ?", ids, 0).Order("step_order asc").Find(&steps)
		stepMap := make(map[int][]models.OpsPipelineStep)
		for _, step := range steps {
			stepMap[step.PipelineId] = append(stepMap[step.PipelineId], step)
		}
		for i := range result.Data {
			result.Data[i].Steps = stepMap[result.Data[i].ID]
		}
	}
	return result, nil
}

// GetByID 根据ID查询编排（含步骤）
func (s *TaskPipelineService) GetByID(id int) (*models.OpsTaskPipeline, error) {
	var pipeline models.OpsTaskPipeline
	if err := models.DB.Where("id = ? AND del_flag = ?", id, 0).First(&pipeline).Error; err != nil {
		return nil, errors.New("编排不存在")
	}
	var steps []models.OpsPipelineStep
	models.DB.Where("pipeline_id = ? AND del_flag = ?", id, 0).Order("step_order asc").Find(&steps)
	pipeline.Steps = steps
	return &pipeline, nil
}

// Add 新增编排
func (s *TaskPipelineService) Add(pipeline *models.OpsTaskPipeline) error {
	if pipeline.Name == "" {
		return errors.New("编排名称不能为空")
	}
	if len(pipeline.Steps) == 0 {
		return errors.New("编排步骤不能为空")
	}
	if err := models.DB.Create(pipeline).Error; err != nil {
		log.Printf("新增编排失败: %v", err)
		return errors.New("新增失败")
	}
	return nil
}

// Edit 编辑编排
func (s *TaskPipelineService) Edit(pipeline *models.OpsTaskPipeline) error {
	if pipeline.ID == 0 {
		return errors.New("编排ID不能为空")
	}
	var existing models.OpsTaskPipeline
	if err := models.DB.Where("id = ? AND del_flag = ?", pipeline.ID, 0).First(&existing).Error; err != nil {
		return errors.New("编排不存在")
	}

	tx := models.DB.Begin()
	// 更新编排基本信息
	if err := tx.Model(&existing).Updates(map[string]interface{}{
		"name":        pipeline.Name,
		"description": pipeline.Description,
	}).Error; err != nil {
		tx.Rollback()
		return errors.New("更新失败")
	}
	// 删除旧步骤
	if err := tx.Where("pipeline_id = ?", pipeline.ID).Delete(&models.OpsPipelineStep{}).Error; err != nil {
		tx.Rollback()
		return errors.New("更新步骤失败")
	}
	// 创建新步骤
	for i := range pipeline.Steps {
		pipeline.Steps[i].PipelineId = pipeline.ID
		pipeline.Steps[i].ID = 0
		if err := tx.Create(&pipeline.Steps[i]).Error; err != nil {
			tx.Rollback()
			return errors.New("更新步骤失败")
		}
	}
	tx.Commit()
	return nil
}

// Delete 删除编排
func (s *TaskPipelineService) Delete(id int) error {
	var existing models.OpsTaskPipeline
	if err := models.DB.Where("id = ? AND del_flag = ?", id, 0).First(&existing).Error; err != nil {
		return errors.New("编排不存在")
	}
	if err := models.DB.Model(&existing).Update("del_flag", 1).Error; err != nil {
		return errors.New("删除失败")
	}
	return nil
}
