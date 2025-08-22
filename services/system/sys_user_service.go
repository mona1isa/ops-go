package system

import (
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/controllers/system/request"
	"github.com/zhany/ops-go/models"
	"log"
)

type UserService struct {
}

// AddUser 新增用户
func (u *UserService) AddUser(request request.UserRequest) error {
	user := models.SysUser{
		DeptId:    request.DeptId,
		UserName:  request.UserName,
		Email:     request.Email,
		Phone:     request.Phone,
		Sex:       request.Sex,
		Avatar:    request.Avatar,
		Password:  request.Password,
		Status:    request.Status,
		LoginIP:   request.LoginIP,
		LoginDate: request.LoginDate,
	}
	if err := config.DB.Create(&user).Error; err != nil {
		log.Println("添加用户失败：", err)
		return err
	}
	return nil
}

func (u *UserService) EditUser(request request.EditUserRequest) error {
	id := request.Id
	user := models.SysUser{}
	find := config.DB.Where("id = ?", id).Find(&user)
	if find.Error != nil {
		log.Println("用户不存在，ID: ", id)
		return find.Error
	}
	tx := config.DB.Model(&models.SysUser{}).Where("id = ?", id).Updates(request)
	if tx.Error != nil {
		log.Println("更新用户失败：", tx.Error)
		return tx.Error
	}
	return nil
}

// 编辑用户

// 用户列表分页

// 全部用户
func (u *UserService) All() ([]models.SysUser, error) {
	var user []models.SysUser
	tx := config.DB.Find(&user)
	if tx.Error != nil {
		log.Println("查询用户失败：", tx.Error)
		return nil, tx.Error
	}
	return user, nil
}

// 删除用户
func (*UserService) Delete(id string) error {
	tx := config.DB.Delete(&models.SysUser{}, id)
	if tx.Error != nil {
		log.Println("删除用户失败：", tx.Error)
		return tx.Error
	}
	return nil
}
