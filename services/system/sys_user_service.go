package system

import (
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/controllers/system/request"
	"github.com/zhany/ops-go/models"
	"golang.org/x/crypto/bcrypt"
	"log"
)

type UserService struct {
}

func (u *UserService) UserLogin(request request.LoginRequest) error {
	username := request.Username
	var user models.SysUser
	dbUser := config.DB.Where("user_name = ?", username).Find(&user)
	if dbUser.Error != nil {
		log.Println("用户不存在：", dbUser.Error)
		return dbUser.Error
	}

	if err := u.CheckHashPassword(request.Password, user.Password); err != nil {
		log.Println("密码错误：", err)
		return err
	}
	return nil
}

// AddUser 新增用户
func (u *UserService) AddUser(request request.UserRequest) error {
	password := request.Password
	hashedPassword, err := u.HashPassword(password)
	if err != nil {
		log.Println("密码加密失败：", err)
		return err
	}
	user := models.SysUser{
		DeptId:    request.DeptId,
		UserName:  request.UserName,
		Email:     request.Email,
		Phone:     request.Phone,
		Sex:       request.Sex,
		Avatar:    request.Avatar,
		Status:    request.Status,
		LoginIP:   request.LoginIP,
		LoginDate: request.LoginDate,
	}
	// 加密存储密码
	user.Password = hashedPassword
	if err := config.DB.Create(&user).Error; err != nil {
		log.Println("添加用户失败：", err)
		return err
	}
	return nil
}

// EditUser 编辑用户
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

// HashPassword 密码加密
func (*UserService) HashPassword(password string) (string, error) {
	fromPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("密码加密失败：", err)
		return "", err
	}
	return string(fromPassword), nil
}

func (*UserService) CheckHashPassword(password string, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		log.Println("密码校验失败：", err)
		return err
	}
	return nil
}
