package instance

import (
	"errors"
	"log"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// TaskTemplateService 任务模板服务
type TaskTemplateService struct{}

// List 分页查询任务模板
func (s *TaskTemplateService) List(pageNum, pageSize int, name string, taskType *int8) (models.PageResult[models.OpsTaskTemplate], error) {
	return models.Paginate[models.OpsTaskTemplate](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		if name != "" {
			db = db.Where("name like ?", "%"+name+"%")
		}
		if taskType != nil {
			db = db.Where("type = ?", *taskType)
		}
		return db.Where("del_flag = ?", 0).Order("id desc")
	})
}

// GetByID 根据ID查询任务模板
func (s *TaskTemplateService) GetByID(id int) (*models.OpsTaskTemplate, error) {
	var template models.OpsTaskTemplate
	if err := models.DB.Where("id = ? AND del_flag = ?", id, 0).First(&template).Error; err != nil {
		return nil, errors.New("模板不存在")
	}
	return &template, nil
}

// Add 新增任务模板
func (s *TaskTemplateService) Add(template *models.OpsTaskTemplate) error {
	if template.Name == "" {
		return errors.New("模板名称不能为空")
	}
	if template.Type < models.TaskTypeCommand || template.Type > models.TaskTypeFile {
		return errors.New("无效的任务类型")
	}
	if template.Timeout <= 0 {
		template.Timeout = 300
	}
	if err := models.DB.Create(template).Error; err != nil {
		log.Printf("新增任务模板失败: %v", err)
		return errors.New("新增失败")
	}
	return nil
}

// Edit 编辑任务模板
func (s *TaskTemplateService) Edit(template *models.OpsTaskTemplate) error {
	if template.ID == 0 {
		return errors.New("模板ID不能为空")
	}
	var existing models.OpsTaskTemplate
	if err := models.DB.Where("id = ? AND del_flag = ?", template.ID, 0).First(&existing).Error; err != nil {
		return errors.New("模板不存在")
	}
	updates := map[string]interface{}{
		"name":        template.Name,
		"type":        template.Type,
		"content":     template.Content,
		"script_lang": template.ScriptLang,
		"src_path":    template.SrcPath,
		"dest_path":   template.DestPath,
		"timeout":     template.Timeout,
		"key_id":      template.KeyId,
		"description": template.Description,
	}
	if err := models.DB.Model(&existing).Updates(updates).Error; err != nil {
		return errors.New("更新失败")
	}
	return nil
}

// Delete 删除任务模板
func (s *TaskTemplateService) Delete(id int) error {
	var existing models.OpsTaskTemplate
	if err := models.DB.Where("id = ? AND del_flag = ?", id, 0).First(&existing).Error; err != nil {
		return errors.New("模板不存在")
	}
	if err := models.DB.Model(&existing).Update("del_flag", 1).Error; err != nil {
		return errors.New("删除失败")
	}
	return nil
}

// InitTaskMenu 初始化任务编排菜单
func InitTaskMenu() {
	// 检查是否已初始化
	var count int64
	models.DB.Model(&models.SysMenu{}).Where("name = ? AND del_flag = ?", "任务编排", "0").Count(&count)
	if count > 0 {
		// 任务编排目录已存在，检查脚本管理子菜单是否存在
		initScriptMenuIfNeeded()
		return
	}

	// 创建"任务编排"目录菜单
	dirMenu := models.SysMenu{
		Name:     "任务编排",
		OrderNum:  5,
		Path:      "/taskOrchestration",
		Component:  "layout/routerView/parent",
		Type:       "M",
		Status:     true,
		Icon:       "ele-SetUp",
		KeepAlive:  true,
	}
	if err := models.DB.Create(&dirMenu).Error; err != nil {
		log.Printf("初始化任务编排菜单失败: %v", err)
		return
	}

	// 创建子菜单
	subMenus := []models.SysMenu{
		{
			Name:       "任务模板",
			ParentId:   dirMenu.ID,
			OrderNum:   1,
			Path:       "/taskOrchestration/taskTemplate",
			Component:  "taskTemplate/index",
			Type:       "C",
			Status:     true,
			Icon:       "ele-Document",
			KeepAlive:  true,
			Perms:      "task:template:list",
		},
		{
			Name:       "任务编排管理",
			ParentId:   dirMenu.ID,
			OrderNum:   2,
			Path:       "/taskOrchestration/taskPipeline",
			Component:  "taskPipeline/index",
			Type:       "C",
			Status:     true,
			Icon:       "ele-Connection",
			KeepAlive:  true,
			Perms:      "task:pipeline:list",
		},
		{
			Name:       "任务执行",
			ParentId:   dirMenu.ID,
			OrderNum:   3,
			Path:       "/taskOrchestration/taskExecution",
			Component:  "taskExecution/index",
			Type:       "C",
			Status:     true,
			Icon:       "ele-VideoPlay",
			KeepAlive:  true,
			Perms:      "task:execution:list",
		},
		{
			Name:       "脚本管理",
			ParentId:   dirMenu.ID,
			OrderNum:   4,
			Path:       "/taskOrchestration/script",
			Component:  "script/index",
			Type:       "C",
			Status:     true,
			Icon:       "ele-DocumentCopy",
			KeepAlive:  true,
			Perms:      "task:script:list",
		},
	}
	for _, menu := range subMenus {
		if err := models.DB.Create(&menu).Error; err != nil {
			log.Printf("初始化子菜单失败 [%s]: %v", menu.Name, err)
		}
	}

	// 将菜单权限赋予管理员角色
	var adminRole models.SysRole
	if err := models.DB.Where("id = ?", 1).First(&adminRole).Error; err == nil {
		for _, menu := range subMenus {
			models.DB.Create(&models.SysRoleMenu{RoleId: adminRole.ID, MenuId: menu.ID})
		}
		models.DB.Create(&models.SysRoleMenu{RoleId: adminRole.ID, MenuId: dirMenu.ID})
	}

	log.Println("任务编排菜单初始化完成")
}

// initScriptMenuIfNeeded 如果脚本管理菜单不存在，则自动添加
func initScriptMenuIfNeeded() {
	var scriptCount int64
	models.DB.Model(&models.SysMenu{}).Where("name = ? AND del_flag = ?", "脚本管理", "0").Count(&scriptCount)
	if scriptCount > 0 {
		return
	}

	// 查找任务编排目录菜单
	var dirMenu models.SysMenu
	if err := models.DB.Where("name = ? AND del_flag = ?", "任务编排", "0").First(&dirMenu).Error; err != nil {
		log.Printf("未找到任务编排目录菜单，跳过初始化脚本管理菜单: %v", err)
		return
	}

	// 创建脚本管理子菜单
	scriptMenu := models.SysMenu{
		Name:       "脚本管理",
		ParentId:   dirMenu.ID,
		OrderNum:   4,
		Path:       "/taskOrchestration/script",
		Component:  "script/index",
		Type:       "C",
		Status:     true,
		Icon:       "ele-DocumentCopy",
		KeepAlive:  true,
		Perms:      "task:script:list",
	}
	if err := models.DB.Create(&scriptMenu).Error; err != nil {
		log.Printf("初始化脚本管理菜单失败: %v", err)
		return
	}

	// 将菜单权限赋予管理员角色
	var adminRole models.SysRole
	if err := models.DB.Where("id = ?", 1).First(&adminRole).Error; err == nil {
		models.DB.Create(&models.SysRoleMenu{RoleId: adminRole.ID, MenuId: scriptMenu.ID})
	}

	log.Println("脚本管理菜单初始化完成")
}
