package system

import (
	"errors"
	"fmt"
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/middleware"
	"github.com/zhany/ops-go/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"strconv"
	"time"
)

type UserService struct {
}

func (u *UserService) UserLogin(request api.LoginRequest) (string, error) {
	// 验证码校验
	captchaService := CaptchaService{}
	capVal := Captcha{
		Uuid: request.Uuid,
		Text: request.Code,
	}
	rs := captchaService.VerifyCaptcha(&capVal)
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

	// 设置过期时间
	now := time.Now()
	exprTime := now.Add(1 * time.Hour)
	expirationTime := exprTime.Unix()
	token := captchaService.GetUuid()
	userId := strconv.Itoa(user.ID)
	deptId := strconv.Itoa(user.DeptId)
	jwt, err := middleware.GenerateJWT(expirationTime, token, userId, deptId, user.UserName)
	if err != nil {
		log.Println("生成Token异常：", err)
		return "", errors.New("生成Token异常")
	}
	// 创建用户Token
	userToken := models.SysUserToken{
		UserId:   user.ID,
		Token:    token,
		ExpireAt: exprTime,
	}
	// 先查询是否存在记录，存在则更新，不存在则创建记录
	var tokenCount int64
	config.DB.Model(models.SysUserToken{}).Where("user_id=?", userId).Count(&tokenCount)
	if tokenCount > 0 {
		if err := config.DB.Model(models.SysUserToken{}).Where("user_id=?", userId).Updates(&userToken).Error; err != nil {
			log.Println("更新用户Token失败：", err)
			return "", err
		}
	} else {
		if err := config.DB.Create(&userToken).Error; err != nil {
			log.Println("创建用户Token失败：", err)
			return "", err
		}
	}

	// 更新用户登录信息
	user.LoginIP = request.LoginIP
	user.LoginDate = request.LoginDate
	config.DB.Updates(&user)
	return jwt, nil
}

// LogOut 退出登录
func (u *UserService) LogOut(tokenString string) {
	err := middleware.InvalidateToken(tokenString)
	if err != nil {
		return
	}
}

// GetUserInfo 获取用户信息
func (u *UserService) GetUserInfo(userId string) (*api.UserInfo, error) {
	user := models.SysUser{}
	if err := config.DB.Model(&models.SysUser{}).Where("id = ?", userId).First(&user).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	userRole := models.SysUserRole{}
	if err := config.DB.Model(&models.SysUserRole{}).Where("user_id = ?", userId).First(&userRole).Error; err != nil {
		return nil, errors.New("用户角色不存在")
	}

	role := models.SysRole{}
	if err := config.DB.Model(&models.SysRole{}).Where("id = ?", userRole.RoleId).First(&role).Error; err != nil {
		return nil, errors.New("角色不存在")
	}

	userInfo := &api.UserInfo{
		Id:       user.ID,
		DeptId:   user.DeptId,
		UserName: user.UserName,
		Nickname: user.NickName,
		Email:    user.Email,
		Phone:    user.Phone,
		Sex:      user.Sex,
		Avatar:   user.Avatar,
		Status:   user.Status,
		RoleId:   userRole.RoleId,
		RoleName: role.Name,
	}

	return userInfo, nil
}

// AddUser 新增用户
func (u *UserService) AddUser(request api.UserRequest) error {
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
	// 保存用户角色
	userRole := models.SysUserRole{
		UserId: user.ID,
		RoleId: request.RoleId,
	}
	if err := config.DB.Create(&userRole).Error; err != nil {
		log.Println("保存用户角色失败：", err)
		return err
	}
	return nil
}

// EditUser 编辑用户
func (u *UserService) EditUser(request api.EditUserRequest) error {
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
func (u *UserService) Page(userRequest *api.PageUserRequest) (models.PageResult[models.SysUser], error) {
	pageNum := userRequest.PageNum
	pageSize := userRequest.PageSize

	var scopes []func(db *gorm.DB) *gorm.DB
	userNmae := userRequest.UserName
	if userNmae != "" {
		scopes = append(scopes, UserNameScope(userNmae))
	}

	pageResult, err := models.Paginate[models.SysUser](config.DB, pageNum, pageSize, scopes...)
	if err != nil {
		log.Println("查询用户列表异常：", err)
		panic(err)
	}

	// 查询用户角色
	userRole, err := GetUserRole()
	if err != nil {
		log.Println("查询用户角色异常：", err)
	}
	for idx, user := range pageResult.Data {
		for _, role := range userRole {
			if user.ID == role.UserID {
				pageResult.Data[idx].RoleName = role.RoleName
			}
		}
	}

	return pageResult, nil
}

func GetUserRole() ([]models.SysUserRoleResult, error) {
	var result []models.SysUserRoleResult
	err := config.DB.Model(&models.SysUser{}).Joins("JOIN sys_user_role ON sys_user_role.user_id = sys_user.id").
		Joins("JOIN sys_role ON sys_role.id = sys_user_role.role_id").Select("sys_user.id AS userId, sys_role.name AS roleName").Scan(&result).Error
	if err != nil {
		return result, err
	}
	return result, nil
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

// ChangeStatus 修改用户状态
func (*UserService) ChangeStatus(request api.UserStatusRequest) error {
	id := request.Id
	config.DB.Model(&models.SysUser{}).Where("id = ?", id).Update("status", request.Status)
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
