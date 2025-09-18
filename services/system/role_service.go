package system

import (
	"errors"
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services"
	"gorm.io/gorm"
	"log"
)

type RoleService struct{}

// Add 添加角色
func (r *RoleService) Add(request *api.RoleRequest) error {
	// 校验角色名称是否存在
	name := request.Name
	var count int64
	config.DB.Model(&models.SysRole{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return errors.New("角色名称已存在")
	}
	// 添加角色
	role := &models.SysRole{
		Name:     request.Name,
		OrderNum: request.OrderNum,
		Status:   request.Status,
	}
	role.Remark = request.Remark
	tx := config.DB.Model(&models.SysRole{}).Create(&role)
	if err := tx.Error; err != nil {
		log.Println("添加角色失败：", err.Error())
		return errors.New("添加角色失败：" + err.Error())
	}
	roleId := role.ID
	// 添加角色权限
	for _, menuId := range request.MenuIds {
		if err := config.DB.Model(&models.SysRoleMenu{}).Create(&models.SysRoleMenu{RoleId: roleId, MenuId: menuId}).Error; err != nil {
			log.Println("添加角色权限失败：", err.Error())
			tx.Rollback()
			return errors.New("添加角色权限失败：" + err.Error())
		}
	}
	return nil
}

// Edit 编辑角色
func (r *RoleService) Edit(request *api.EditRoleRequest) error {
	id := request.Id
	var count int64
	config.DB.Model(&models.SysRole{}).Where("id = ?", id).Count(&count)
	if count == 0 {
		return errors.New("角色不存在")
	}
	// 编辑角色
	role := &models.SysRole{
		Name:     request.Name,
		OrderNum: request.OrderNum,
		Status:   request.Status,
	}
	role.Remark = request.Remark
	if err := config.DB.Model(&models.SysRole{}).Where("id = ?", id).Updates(role).Error; err != nil {
		log.Println("编辑角色失败：", err.Error())
		return errors.New("编辑角色失败：" + err.Error())
	}
	return nil
}

// List 角色列表
func (r *RoleService) List() ([]models.SysRole, error) {
	all, err := services.FindAll[models.SysRole]()
	if err != nil {
		log.Println("查询角色异常：", err)
		return nil, err
	}
	return all, nil
}

// Page 分页查询角色
func (r *RoleService) Page(roleRequest *api.PageRoleRequest) (models.PageResult[models.SysRole], error) {
	pageNum := roleRequest.PageNum
	pageSize := roleRequest.PageSize

	var scopes []func(db *gorm.DB) *gorm.DB
	if roleRequest.Name != "" {
		scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
			return db.Where("name like ?", "%"+roleRequest.Name+"%")
		})
	}
	if roleRequest.Status != "" {
		scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", roleRequest.Status)
		})
	}

	// 根据 orderNum 排序
	scopes = append(scopes, func(db *gorm.DB) *gorm.DB { return db.Order("order_num asc") })

	pageResult, err := models.Paginate[models.SysRole](config.DB, pageNum, pageSize, scopes...)
	if err != nil {
		panic(err)
	}
	return pageResult, nil
}

// Remove 删除角色
func (r *RoleService) Remove(id int) error {
	if err := config.DB.Delete(&models.SysRole{}, id).Error; err != nil {
		return errors.New("角色删除失败: " + err.Error())
	}
	return nil
}
