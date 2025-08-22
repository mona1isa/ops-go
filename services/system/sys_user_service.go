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

// 编辑用户

// 用户列表分页

// 全部用户

// 删除用户
