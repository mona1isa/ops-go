package system

import (
	"errors"
	"fmt"
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/controllers/system/request"
	"github.com/zhany/ops-go/middleware"
	"github.com/zhany/ops-go/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"strconv"
)

type UserService struct {
}

func (u *UserService) UserLogin(request request.LoginRequest) (string, error) {
	// 验证码校验
	captchaService := CaptchaService{}
	cap := Captcha{
		Uuid: request.Uuid,
		Text: request.Code,
	}
	rs := captchaService.VerifyCaptcha(&cap)
	if !rs {
		return "", errors.New("验证码错误")
	}
	// 验证用户信息
	username := request.Username
	var user models.SysUser
	config.DB.First(&user, "user_name=?", username)
	if user.UserName == "" && user.UserName == username {
		return "", errors.New("用户不存在")
	}

	if err := u.CheckHashPassword(request.Password, user.Password); err != nil {
		log.Println("密码错误：", err)
		return "", errors.New("密码错误")
	}

	userId := strconv.Itoa(int(user.ID))
	deptId := strconv.Itoa(int(user.DeptId))
	jwt, err := middleware.GenerateJWT(userId, deptId, user.UserName)
	if err != nil {
		log.Println("生成Token异常：", err)
		return "", errors.New("生成Token异常")
	}

	// 更新用户登录信息
	user.LoginIP = request.LoginIP
	user.LoginDate = request.LoginDate
	config.DB.Updates(&user)
	return jwt, nil
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
		DeptId:   request.DeptId,
		UserName: request.UserName,
		NickName: request.Nickname,
		Email:    request.Email,
		Phone:    request.Phone,
		Sex:      request.Sex,
		Avatar:   request.Avatar,
		Status:   request.Status,
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
	var count int64
	user := models.SysUser{}
	config.DB.Model(&user).Where("id = ?", id).Count(&count)
	if count == 0 {
		log.Println("用户不存在，ID: ", id)
		return errors.New("用户不存在")
	}
	result := config.DB.Model(&models.SysUser{}).Where("id = ?", id).Updates(models.SysUser{
		DeptId:   request.DeptId,
		UserName: request.UserName,
		NickName: request.Nickname,
		Email:    request.Email,
		Phone:    request.Phone,
		Sex:      request.Sex,
		Avatar:   request.Avatar,
		Status:   request.Status,
	})
	if result.RowsAffected == 0 {
		log.Println("更新用户失败：", result.Error)
		errInfo := fmt.Sprintf("更新用户失败：%s", result.Error)
		return errors.New(errInfo)
	}
	return nil
}

// Page 用户列表分页
func (u *UserService) Page(userRequest *request.PageUserRequest) (models.PageResult[models.SysUser], error) {
	pageNum := userRequest.PageNum
	pageSize := userRequest.PageSize

	var scopes []func(db *gorm.DB) *gorm.DB
	userNmae := userRequest.UserName
	if userNmae != "" {
		scopes = append(scopes, UserNameScope(userNmae))
	}

	pageResult, err := models.Paginate[models.SysUser](config.DB, pageNum, pageSize, scopes...)
	if err != nil {
		panic(err)
	}
	return pageResult, nil
}

// Delete 删除用户
func (*UserService) Delete(id string) error {
	tx := config.DB.Delete(&models.SysUser{}, id)
	if tx.Error != nil {
		log.Println("删除用户失败：", tx.Error)
		return errors.New("删除用户失败")
	}
	return nil
}

// HashPassword 密码加密
func (*UserService) HashPassword(password string) (string, error) {
	fromPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("密码加密失败：", err)
		return "", errors.New("密码加密失败")
	}
	return string(fromPassword), nil
}

func (*UserService) CheckHashPassword(password string, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		log.Println("密码校验失败：", err)
		return errors.New("密码校验失败")
	}
	return nil
}

func UserNameScope(userName string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_name = ?", userName)
	}
}
