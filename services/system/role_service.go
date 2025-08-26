package system

import (
	"errors"
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/controllers/system/request"
	"github.com/zhany/ops-go/models"
	"log"
)

type RoleService struct{}

// Add 添加角色
func (r *RoleService) Add(request *request.RoleRequest) error {
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
	if err := config.DB.Create(role).Error; err != nil {
		log.Println("添加角色失败：", err.Error())
		return errors.New("添加角色失败：" + err.Error())
	}
	return nil
}

// Edit 编辑角色
func (r *RoleService) Edit(request *request.RoleRequest) error {
	return nil
}

// Page 分页查询角色
func (r *RoleService) Page(roleRequest *request.RoleRequest) error {
	return nil
}

// Remove 删除角色
func (r *RoleService) Remove(roleRequest *request.RoleRequest) error {
	return nil
}
