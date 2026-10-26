package system

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/mfa"
	"github.com/zhany/ops-go/middleware"
	"github.com/zhany/ops-go/models"
	logService "github.com/zhany/ops-go/services/logs"
	"github.com/zhany/ops-go/utils"
	"github.com/zhany/ops-go/utils/totp"
)

// MFA 相关时间参数
const (
	mfaBindTTL    = 10 * time.Minute // 绑定会话（未确认的 secret）有效期
	mfaPreTTL     = 5 * time.Minute  // 登录预票据有效期
	mfaMaxAttempt = 5                // 单张票据最大尝试次数
	mfaIssuer     = "Ops-Go"         // otpauth 发行方标识
)

// MfaService TOTP 绑定/校验/登录二次验证
type MfaService struct{}

// IsMfaRequired 是否强制所有用户启用 MFA（环境变量 MFA_REQUIRED）
func IsMfaRequired() bool { return mfa.IsRequired() }

func mfaBindKey(userId int) string {
	return fmt.Sprintf("mfa_bind:%d", userId)
}

func mfaPreKey(preToken string) string {
	return fmt.Sprintf("mfa_pre:%s", preToken)
}

// BeginBind 生成候选密钥存入 Redis（确认前不落库），返回密钥与扫码 URI
func (s *MfaService) BeginBind(userId int) (secret string, otpauthURL string, err error) {
	secret, err = totp.GenerateSecret()
	if err != nil {
		return
	}
	if err = utils.SetCache(mfaBindKey(userId), secret, mfaBindTTL); err != nil {
		return "", "", errors.New("缓存服务不可用，请稍后重试")
	}
	var user models.SysUser
	if err = models.DB.First(&user, userId).Error; err != nil {
		utils.DelCache(mfaBindKey(userId))
		return "", "", errors.New("用户不存在")
	}
	return secret, totp.ProvisioningURL(secret, user.UserName, mfaIssuer), nil
}

// ConfirmBind 校验动态码并把密钥与恢复码落库；返回一次性恢复码明文（仅此一次）
func (s *MfaService) ConfirmBind(userId int, code string) (recoveryCodes []string, err error) {
	secret, err := utils.GetCache(mfaBindKey(userId))
	if err != nil || secret == "" {
		return nil, errors.New("绑定会话已过期，请重新发起绑定")
	}
	if !totp.Verify(secret, code, time.Now()) {
		return nil, errors.New("动态码不正确，请检查认证器时间与输入")
	}

	encSecret, err := utils.EncryptPassword(secret)
	if err != nil {
		log.Println("加密 TOTP 密钥失败：", err)
		return nil, errors.New("加密密钥失败")
	}
	recoveryCodes, err = totp.GenerateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}
	hashes, err := mfa.HashRecoveryCodes(recoveryCodes)
	if err != nil {
		return nil, err
	}
	hashJSON, err := json.Marshal(hashes)
	if err != nil {
		return nil, err
	}

	if err = models.DB.Model(&models.SysUser{}).Where("id = ?", userId).Updates(map[string]any{
		"totp_secret":    encSecret,
		"totp_enabled":   1,
		"recovery_codes": string(hashJSON),
	}).Error; err != nil {
		log.Println("保存 MFA 绑定失败：", err)
		return nil, errors.New("保存绑定信息失败")
	}
	utils.DelCache(mfaBindKey(userId))
	return recoveryCodes, nil
}

// Unbind 校验当前密码后清空 MFA 绑定（密码为 RSA 加密 + base64，与修改密码一致）
func (s *MfaService) Unbind(userId int, encryptedPassword string) error {
	var user models.SysUser
	if err := models.DB.First(&user, userId).Error; err != nil {
		return errors.New("用户不存在")
	}
	raw, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return errors.New("解码密码失败")
	}
	privateKey, err := utils.ParsePrivateKeyFromPEM(utils.PrivateKey)
	if err != nil {
		return errors.New("解析私钥失败")
	}
	password, err := utils.Decrypt(privateKey, raw)
	if err != nil {
		return errors.New("密码解密失败")
	}
	userSvc := UserService{}
	if err := userSvc.CheckHashPassword(string(password), user.Password); err != nil {
		return errors.New("密码错误")
	}

	return models.DB.Model(&models.SysUser{}).Where("id = ?", userId).Updates(map[string]any{
		"totp_secret":    "",
		"totp_enabled":   0,
		"recovery_codes": "",
	}).Error
}

// Status 当前用户 MFA 状态（供个人中心展示与登录后引导）
func (s *MfaService) Status(userId int) (enabled bool, required bool) {
	var user models.SysUser
	if err := models.DB.Select("totp_enabled").First(&user, userId).Error; err != nil {
		return false, IsMfaRequired()
	}
	return user.TotpEnabled == 1, IsMfaRequired()
}

// CreatePreToken 密码校验通过后签发一次性预票据（存 Redis，值带尝试计数）
func (s *MfaService) CreatePreToken(userId int) (string, error) {
	preToken := strings.ReplaceAll(uuid.New().String(), "-", "")
	if err := utils.SetCache(mfaPreKey(preToken), strconv.Itoa(userId)+":0", mfaPreTTL); err != nil {
		return "", errors.New("缓存服务不可用，请稍后重试")
	}
	return preToken, nil
}

// VerifyPreToken 校验预票据与动态码/恢复码；成功作废票据并返回用户，失败累加尝试次数。
// user 与 err 可能同时非空：票据能定位到账号但校验失败时也返回 user，供登录日志记录身份。
func (s *MfaService) VerifyPreToken(preToken, code, recovery string) (*models.SysUser, error) {
	key := mfaPreKey(preToken)
	val, err := utils.GetCache(key)
	if err != nil || val == "" {
		return nil, errors.New("登录会话已过期，请重新登录")
	}
	parts := strings.SplitN(val, ":", 2)
	userId, err := strconv.Atoi(parts[0])
	if err != nil || userId <= 0 {
		utils.DelCache(key)
		return nil, errors.New("登录票据异常，请重新登录")
	}
	attempts := 0
	if len(parts) > 1 {
		attempts, _ = strconv.Atoi(parts[1])
	}

	var user models.SysUser
	if err := models.DB.First(&user, userId).Error; err != nil || user.TotpEnabled != 1 {
		utils.DelCache(key)
		return nil, errors.New("用户状态异常，请重新登录")
	}
	if attempts >= mfaMaxAttempt {
		utils.DelCache(key)
		return &user, errors.New("验证失败次数过多，请重新登录")
	}

	ok := false
	usedRecovery := strings.TrimSpace(recovery) != ""
	if usedRecovery {
		ok = mfa.ConsumeRecoveryCode(&user, recovery)
	} else if strings.TrimSpace(code) != "" {
		ok = mfa.VerifyTotpCode(&user, code)
	}
	if !ok {
		attempts++
		if attempts >= mfaMaxAttempt {
			utils.DelCache(key)
			return &user, errors.New("验证失败次数过多，请重新登录")
		}
		utils.SetCache(key, fmt.Sprintf("%d:%d", userId, attempts), mfaPreTTL)
		if usedRecovery {
			return &user, errors.New("恢复码无效或已被使用")
		}
		return &user, errors.New("动态验证码错误")
	}

	utils.DelCache(key)
	return &user, nil
}

// IssueLoginToken 密码或 MFA 验证通过后的统一签发：JWT + sys_user_token 单条覆盖 + 登录信息更新
func IssueLoginToken(user models.SysUser, loginIP string) (string, error) {
	now := time.Now()
	exprTime := now.Add(1 * time.Hour)
	expirationTime := exprTime.Unix()
	captchaSvc := CaptchaService{}
	token := captchaSvc.GetUuid()
	userId := strconv.Itoa(user.ID)
	deptId := strconv.Itoa(user.DeptId)
	jwt, err := middleware.GenerateJWT(expirationTime, token, userId, deptId, user.UserName)
	if err != nil {
		log.Println("生成Token异常：", err)
		return "", errors.New("生成Token异常")
	}
	userToken := models.SysUserToken{
		UserId:   user.ID,
		Token:    token,
		ExpireAt: exprTime,
	}
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
	now = time.Now()
	user.LoginIP = loginIP
	user.LoginDate = &now
	user.UpdateLoginInfo()
	return jwt, nil
}

// MfaLogin 登录第二步：校验动态码/恢复码并签发 token。
// 两步都要留登录日志：第一步只写「密码通过，等待MFA验证」的待验证中间态，
// 成功与失败都在此闭环，否则启用 MFA 的账号在行为分析里永远只有中间态记录。
func (s *MfaService) MfaLogin(request api.MfaLoginRequest, loginIP, userAgent string) (string, error) {
	user, err := s.VerifyPreToken(request.PreToken, request.Code, request.Recovery)
	if user == nil {
		// 票据已失效或定位不到账号，没有身份可留痕
		return "", err
	}
	submitted := request.Code
	if strings.TrimSpace(request.Recovery) != "" {
		submitted = request.Recovery
	}
	result := int8(models.LoginResultSuccess)
	reason := ""
	if err != nil {
		result = int8(models.LoginResultFail)
		reason = err.Error()
	} else {
		submitted = "" // 校验通过的动态码不留任何特征
	}
	logService.RecordLogin(logService.LoginLogEntry{
		UserName:  user.UserName,
		UserId:    user.ID,
		DeptId:    user.DeptId,
		Result:    result,
		Reason:    reason,
		Ip:        loginIP,
		UserAgent: userAgent,
		Platform:  models.LoginSourceWeb,
		Password:  submitted,
	})
	if err != nil {
		return "", err
	}
	return IssueLoginToken(*user, loginIP)
}
