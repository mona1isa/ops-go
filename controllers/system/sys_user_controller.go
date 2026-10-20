package system

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/middleware"
	"github.com/zhany/ops-go/models"
	logService "github.com/zhany/ops-go/services/logs"
	"github.com/zhany/ops-go/services/system"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SysUserController struct {
	controllers.BaseController
}

// LoginHandler 登录
func (s *SysUserController) LoginHandler(ctx *gin.Context) {
	loginRequest := api.LoginRequest{}
	err := ctx.ShouldBindJSON(&loginRequest)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置登录IP和时间
	loginRequest.LoginIP = ctx.ClientIP()
	loginRequest.UserAgent = ctx.Request.UserAgent()
	now := time.Now()
	loginRequest.LoginDate = &now

	service := system.UserService{}
	result, err := service.UserLogin(loginRequest)
	if err != nil {
		msg := gin.H{
			"code": http.StatusBadRequest,
			"msg":  err.Error(),
		}
		ctx.JSON(http.StatusBadRequest, msg)
		return
	}
	resp := map[string]any{
		"code": 200,
		"msg":  "success",
	}
	if result.Token != "" {
		resp["token"] = result.Token
	}
	if result.MfaRequired {
		resp["mfaRequired"] = true
		resp["preToken"] = result.PreToken
	}
	if result.BindRequired {
		resp["bindRequired"] = true
	}
	ctx.JSON(http.StatusOK, resp)
}

// MfaLoginHandler 登录第二步：动态验证码 / 恢复码
func (s *SysUserController) MfaLoginHandler(ctx *gin.Context) {
	request := api.MfaLoginRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": err.Error()})
		return
	}
	request.PreToken = strings.TrimSpace(request.PreToken)

	service := system.MfaService{}
	jwt, err := service.MfaLogin(request, ctx.ClientIP())
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code":  200,
		"msg":   "success",
		"token": jwt,
	})
}

// MfaStatusHandler 当前用户 MFA 状态
func (s *SysUserController) MfaStatusHandler(ctx *gin.Context) {
	mfaSvc := system.MfaService{}
	userId, _ := strconv.Atoi(s.GetUserId(ctx))
	enabled, required := mfaSvc.Status(userId)
	s.Success(ctx, gin.H{"enabled": enabled, "required": required})
}

// MfaBindStartHandler 发起绑定：返回密钥与 otpauth URI
func (s *SysUserController) MfaBindStartHandler(ctx *gin.Context) {
	mfaSvc := system.MfaService{}
	userId, _ := strconv.Atoi(s.GetUserId(ctx))
	if userId <= 0 {
		s.Failure(ctx, http.StatusBadRequest, "用户未登录")
		return
	}
	secret, url, err := mfaSvc.BeginBind(userId)
	if err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.Success(ctx, gin.H{"secret": secret, "otpauthUrl": url})
}

// MfaBindConfirmHandler 确认绑定：校验动态码并返回一次性恢复码
func (s *SysUserController) MfaBindConfirmHandler(ctx *gin.Context) {
	request := api.MfaConfirmRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	mfaSvc := system.MfaService{}
	userId, _ := strconv.Atoi(s.GetUserId(ctx))
	if userId <= 0 {
		s.Failure(ctx, http.StatusBadRequest, "用户未登录")
		return
	}
	codes, err := mfaSvc.ConfirmBind(userId, request.Code)
	if err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.Success(ctx, gin.H{"recoveryCodes": codes})
}

// MfaUnbindHandler 解绑：验证当前密码后清除 MFA 设置
func (s *SysUserController) MfaUnbindHandler(ctx *gin.Context) {
	request := api.MfaUnbindRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	mfaSvc := system.MfaService{}
	userId, _ := strconv.Atoi(s.GetUserId(ctx))
	if userId <= 0 {
		s.Failure(ctx, http.StatusBadRequest, "用户未登录")
		return
	}
	if err := mfaSvc.Unbind(userId, request.Password); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// LogOutHandler 登出
func (s *SysUserController) LogOutHandler(ctx *gin.Context) {
	service := system.UserService{}
	tokenString := ctx.GetHeader("Authorization")

	// /api/user/logout 属于免鉴权路径，需要从 Token 反查身份再留痕
	if claims, err := middleware.ValidateToken(tokenString); err == nil {
		userId, _ := strconv.Atoi(claims.UserID)
		deptId, _ := strconv.Atoi(claims.DeptId)
		logService.RecordLogin(logService.LoginLogEntry{
			UserName:  claims.Username,
			UserId:    userId,
			DeptId:    deptId,
			Result:    models.LoginResultLogout,
			Reason:    "用户主动退出",
			Ip:        ctx.ClientIP(),
			UserAgent: ctx.Request.UserAgent(),
			Platform:  models.LoginSourceWeb,
		})
	}

	service.LogOut(tokenString)
	s.JustSuccess(ctx)
}

// AddUserHandler 添加用户
func (s *SysUserController) AddUserHandler(ctx *gin.Context) {
	userRequest := api.UserRequest{}
	if err := ctx.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userId := s.GetUserId(ctx)
	userRequest.CreateBy = userId
	userRequest.UpdateBy = userId
	userService := system.UserService{}
	if err := userService.AddUser(userRequest); err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// UserInfoHandler 获取用户信息
func (s *SysUserController) UserInfoHandler(ctx *gin.Context) {
	userId := s.GetUserId(ctx)
	userService := system.UserService{}
	userInfo, err := userService.GetUserInfo(userId)
	if err != nil {
		s.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	s.Success(ctx, userInfo)
}

// UpdatePersonalInfo 更新个人信息
func (s *SysUserController) UpdatePersonalInfo(ctx *gin.Context) {
	userRequest := api.EditUserRequest{}
	if err := ctx.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userRequest.UpdateBy = s.GetUserId(ctx)
	service := system.UserService{}
	if err := service.UpdatePersonalInfo(userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// UpdatePassword 更新用户密码
func (s *SysUserController) UpdatePassword(ctx *gin.Context) {
	userRequest := api.UpdatePasswordRequest{}
	if err := ctx.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	service := system.UserService{}
	if err := service.UpdatePassword(userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

// EditUserHandler 编辑用户
func (s *SysUserController) EditUserHandler(ctx *gin.Context) {
	userRequest := api.EditUserRequest{}
	if err := ctx.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if userRequest.Id <= 0 {
		s.Failure(ctx, http.StatusBadRequest, errors.New("id is required"))
		return
	}

	userRequest.UpdateBy = s.GetUserId(ctx)
	service := system.UserService{}
	if err := service.EditUser(userRequest); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.JustSuccess(ctx)
}

func (s *SysUserController) Page(c *gin.Context) {
	userRequest := api.PageUserRequest{}
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		s.Failure(c, http.StatusBadRequest, err)
		return
	}
	service := system.UserService{}
	all, err := service.Page(&userRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	s.Success(c, all)
}

func (s *SysUserController) Delete(ctx *gin.Context) {
	service := system.UserService{}
	id := ctx.Param("id")
	if err := service.Delete(id); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
	}
	s.JustSuccess(ctx)
}

// ChangeStatus 修改用户状态
func (s *SysUserController) ChangeStatus(ctx *gin.Context) {
	userStatus := api.UserStatusRequest{}
	if err := ctx.ShouldBindJSON(&userStatus); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	id := userStatus.Id
	if id == controllers.AdminUserId {
		s.Failure(ctx, http.StatusBadRequest, "不能修改管理员状态")
		return
	}

	service := system.UserService{}
	if err := service.ChangeStatus(userStatus); err != nil {
		s.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}
	s.JustSuccess(ctx)
}
