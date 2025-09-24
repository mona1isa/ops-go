package system

import (
	"errors"
	"fmt"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/middleware"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
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
	models.DB.First(&user, "user_name=?", username)
	if user.UserName == "" && user.UserName == username {
		return "", errors.New("用户不存在")
	}

	// 用户被禁用后不准登录
	if strings.EqualFold(user.Status, "0") {
		return "", errors.New("用户被禁用，请联系管理员")
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
	models.DB.Model(models.SysUserToken{}).Where("user_id=?", userId).Count(&tokenCount)
	if tokenCount > 0 {
		if err := models.DB.Model(models.SysUserToken{}).Where("user_id=?", userId).Updates(&userToken).Error; err != nil {
			log.Println("更新用户Token失败：", err)
			return "", err
		}
	} else {
		if err := models.DB.Create(&userToken).Error; err != nil {
			log.Println("创建用户Token失败：", err)
			return "", err
		}
	}

	// 更新用户登录信息
	user.LoginIP = request.LoginIP
	user.LoginDate = request.LoginDate
	models.DB.Updates(&user)
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
	if err := models.DB.Model(&models.SysUser{}).Where("id = ?", userId).First(&user).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	roleIds := make([]int, 0)
	var roleNames string
	userRole, err := GetUserRole()
	if err != nil {
		log.Println("查询用户角色异常：", err)
	}
	for _, v := range userRole {
		if v.UserId == user.ID {
			roleIds = append(roleIds, v.RoleId)
			roleNames += v.RoleName + ","
		}
	}

	userInfo := &api.UserInfo{
		Id:        user.ID,
		DeptId:    user.DeptId,
		UserName:  user.UserName,
		Nickname:  user.NickName,
		Email:     user.Email,
		Phone:     user.Phone,
		Sex:       user.Sex,
		Avatar:    user.Avatar,
		Status:    user.Status,
		RoleIds:   roleIds,
		RoleNames: roleNames,
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
		RoleIds:  request.RoleIds,
	}
	// 加密存储密码
	user.Password = hashedPassword
	if err := models.DB.Create(&user).Error; err != nil {
		log.Println("添加用户失败：", err)
		return err
	}
	// 保存用户角色移动到 Hook 中执行

	return nil
}

// EditUser 编辑用户
func (u *UserService) EditUser(request api.EditUserRequest) error {
	id := request.Id
	var user models.SysUser
	if err := models.DB.Where("id = ?", id).First(&user).Error; err != nil {
		log.Println("用户不存在，ID: ", id)
		return errors.New("用户不存在")
	}
	user.DeptId = request.DeptId
	user.UserName = request.UserName
	user.NickName = request.Nickname
	user.Email = request.Email
	user.Phone = request.Phone
	user.Sex = request.Sex
	user.Avatar = request.Avatar
	user.Status = request.Status
	user.RoleIds = request.RoleIds
	result := models.DB.Save(&user)
	if result.RowsAffected == 0 {
		log.Println("更新用户失败：", result.Error)
		errInfo := fmt.Sprintf("更新用户失败：%s", result.Error)
		return errors.New(errInfo)
	}
	// 修改角色逻辑移动到Hook中执行

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

	pageResult, err := models.Paginate[models.SysUser](models.DB, pageNum, pageSize, scopes...)
	if err != nil {
		log.Println("查询用户列表异常：", err)
		panic(err)
	}

	// 查询部门信息
	sysDepts, err := services.FindAll[models.SysDept]()

	// 查询用户角色
	userRole, err := GetUserRole()
	if err != nil {
		log.Println("查询用户角色异常：", err)
	}
	for idx, user := range pageResult.Data {
		// 赋值部门信息
		for _, dept := range sysDepts {
			if user.DeptId == dept.ID {
				pageResult.Data[idx].DeptName = dept.Name
			}
		}
		// 赋值角色信息
		roleNames := ""
		roleIds := make([]int, 0)
		for _, role := range userRole {
			if user.ID == role.UserId {
				roleNames = roleNames + role.RoleName + ","
				roleIds = append(roleIds, role.RoleId)
			}
		}
		roleNames = strings.TrimRight(roleNames, ",")
		pageResult.Data[idx].RoleNames = roleNames
		pageResult.Data[idx].RoleIds = roleIds
	}

	return pageResult, nil
}

func GetUserRole() ([]models.SysUserRoleResult, error) {
	var result []models.SysUserRoleResult
	err := models.DB.Model(&models.SysUser{}).Joins("JOIN sys_user_role ON sys_user_role.user_id = sys_user.id").
		Joins("JOIN sys_role ON sys_role.id = sys_user_role.role_id").Select("sys_user.id AS userId, sys_role.name AS roleName, sys_role.id AS roleId").Scan(&result).Error
	if err != nil {
		return result, err
	}
	return result, nil
}

// Delete 删除用户
func (*UserService) Delete(id string) error {
	tx := models.DB.Delete(&models.SysUser{}, id)
	if tx.Error != nil {
		log.Println("删除用户失败：", tx.Error)
		return errors.New("删除用户失败")
	}
	// 删除用户关联角色信息
	models.DB.Model(&models.SysUserRole{}).Where("user_id = ?", id).Delete(&models.SysUserRole{})
	return nil
}

// ChangeStatus 修改用户状态
func (*UserService) ChangeStatus(request api.UserStatusRequest) error {
	id := request.Id
	models.DB.Model(&models.SysUser{}).Where("id = ?", id).Update("status", request.Status)
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
		return db.Where("user_name like ?", "%"+userName+"%")
	}
}
