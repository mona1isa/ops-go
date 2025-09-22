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

const DISABLED = "0"

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
	// 保存角色菜单
	_ = saveRoleMenu(request.MenuIds, id)
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

// GetMenuIds 获取角色菜单
func (r *RoleService) GetMenuIds(roleId int) []int {
	var menuIds []int
	config.DB.Model(models.SysRoleMenu{}).Select("menu_id").Where("role_id = ?", roleId).Find(&menuIds)
	return menuIds
}

// GetUserIds 获取角色用户
func (r *RoleService) GetUserIds(roleId int) []int {
	var userIds []int
	config.DB.Model(models.SysUserRole{}).Select("user_id").Where("role_id = ?", roleId).Find(&userIds)
	return userIds
}

// RoleAsignUsers 角色授权
func (r *RoleService) RoleAsignUsers(request api.RoleAsignRequest) error {
	roleId := request.RoleId
	var role models.SysRole
	if err := config.DB.Model(models.SysRole{}).Where("id = ?", roleId).Find(&role).Error; err != nil {
		return errors.New("角色不存在")
	}

	if role.Status == DISABLED {
		return errors.New("角色已禁用，无法授权")
	}

	_ = saveUserRole(request.UserIds, roleId)
	return nil
}

// saveRoleMenu 保存角色菜单
func saveRoleMenu(menuIds []int, roleId int) error {
	if len(menuIds) == 0 {
		return nil
	}
	config.DB.Model(&models.SysRoleMenu{}).Where("role_id = ?", roleId).Delete(&models.SysRoleMenu{})
	for _, menuId := range menuIds {
		if err := config.DB.Model(&models.SysRoleMenu{}).Create(&models.SysRoleMenu{RoleId: roleId, MenuId: menuId}).Error; err != nil {
			return errors.New("保存角色菜单失败: " + err.Error())
		}
	}
	return nil
}

// saveUserRole 保存用户角色
func saveUserRole(userIds []int, roleId int) error {
	if len(userIds) == 0 {
		return nil
	}

	// 删除旧的角色用户
	config.DB.Model(&models.SysUserRole{}).Where("role_id = ?", roleId).Delete(&models.SysUserRole{})

	// 添加新的角色用户
	for _, userId := range userIds {
		config.DB.Create(&models.SysUserRole{RoleId: roleId, UserId: userId})
	}
	return nil
}
