// Package mfa 提供 Web 登录与 SSH 堡垒机共用的 TOTP/恢复码校验逻辑。
// 独立成包是为了打破 bastion 与 services/system 之间的循环导入
// （services/system/dashboard_service.go 已引用 bastion）。
package mfa

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/utils"
	"github.com/zhany/ops-go/utils/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	totpCodeLen   = 6  // TOTP 动态码位数
	recoveryPlain = 16 // 恢复码去连字符后的长度（格式 XXXXXXXX-XXXXXXXX）

	// SecondStagePrefix 堡垒机内部标记：带此前缀的口令只走二次验证码校验，不再比对密码
	SecondStagePrefix = "#mfa#"
)
// IsRequired 是否强制所有用户启用 MFA（环境变量 MFA_REQUIRED）
func IsRequired() bool {
	val := os.Getenv("MFA_REQUIRED")
	return strings.EqualFold(val, "true") || val == "1"
}

// VerifyTotpCode 解密用户 TOTP 密钥并校验 6 位动态码
func VerifyTotpCode(user *models.SysUser, code string) bool {
	plain, err := utils.DecryptKey(user.TotpSecret)
	if err != nil {
		log.Println("解密 TOTP 密钥失败：", err)
		return false
	}
	return totp.Verify(plain, code, time.Now())
}

// NormalizeRecoveryCode 去除分隔符与大小写差异，容忍用户手输格式
func NormalizeRecoveryCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
}

// HashRecoveryCodes 生成恢复码的 bcrypt 哈希列表
func HashRecoveryCodes(codes []string) ([]string, error) {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		h, err := bcrypt.GenerateFromPassword([]byte(NormalizeRecoveryCode(c)), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		out = append(out, string(h))
	}
	return out, nil
}

// ParseRecoveryHashes 解析库中存储的恢复码哈希 JSON 数组
func ParseRecoveryHashes(stored string) ([]string, error) {
	if strings.TrimSpace(stored) == "" {
		return nil, nil
	}
	var hashes []string
	if err := json.Unmarshal([]byte(stored), &hashes); err != nil {
		return nil, err
	}
	return hashes, nil
}

// ConsumeRecoveryCode 恢复码逐个 bcrypt 比对，命中后删除该条（一次性）
func ConsumeRecoveryCode(user *models.SysUser, input string) bool {
	hashes, err := ParseRecoveryHashes(user.RecoveryCodes)
	if err != nil || len(hashes) == 0 {
		return false
	}
	input = NormalizeRecoveryCode(input)
	for i, h := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(input)) == nil {
			remaining := make([]string, 0, len(hashes)-1)
			remaining = append(remaining, hashes[:i]...)
			remaining = append(remaining, hashes[i+1:]...)
			encoded, err := json.Marshal(remaining)
			if err != nil {
				return false
			}
			if err := models.DB.Model(&models.SysUser{}).Where("id = ?", user.ID).Update("recovery_codes", string(encoded)).Error; err != nil {
				log.Println("更新恢复码失败：", err)
			}
			return true
		}
	}
	return false
}

// VerifyCode 校验动态码或恢复码（恢复码命中即消耗）
func VerifyCode(user *models.SysUser, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	if len(code) == totpCodeLen {
		return VerifyTotpCode(user, code)
	}
	return ConsumeRecoveryCode(user, code)
}

// CheckPassword 校验登录密码，bcrypt 比对
func CheckPassword(pass, hashed string) bool {
	if pass == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(pass)) == nil
}

// Candidate 密码+验证码拼接输入按某个切分点得到的组合
type Candidate struct {
	Password string
	Code     string
}

// ConcatCandidates 枚举「密码+动态码/恢复码」拼接输入的所有切分候选。
// SSH password 协议无法分段追问，动态码固定 6 位数字、恢复码 16/17 位，
// 先按最长后缀（带连字符恢复码 17 位）切分，避免合法组合被更短的错误切分抢先；
// 无法形成候选前缀（密码为空）时返回空列表。
func ConcatCandidates(raw string) []Candidate {
	suffixLen := make([]int, 0, 3)
	if len(raw) >= recoveryPlain+1 {
		suffixLen = append(suffixLen, recoveryPlain+1)
	}
	if len(raw) >= recoveryPlain {
		suffixLen = append(suffixLen, recoveryPlain)
	}
	if len(raw) >= totpCodeLen {
		suffixLen = append(suffixLen, totpCodeLen)
	}

	seen := make(map[string]bool)
	var out []Candidate
	for _, n := range suffixLen {
		prefix := raw[:len(raw)-n]
		if prefix == "" || seen[prefix] {
			continue
		}
		seen[prefix] = true
		out = append(out, Candidate{Password: prefix, Code: raw[len(raw)-n:]})
	}
	return out
}

// Result 堡垒机口令认证结果
type Result struct {
	OK     bool
	Reason string // 失败原因，可直接回显/记日志
}

// AuthBastionPassword 堡垒机口令认证：密码校验 + 可选的拼接式二次验证。
// Web 端已绑定 MFA 的用户必须输入「密码+6位动态码」或「密码+恢复码」；
// 未绑定时按全局强制策略（MFA_REQUIRED）决定是否放行。
func AuthBastionPassword(user *models.SysUser, raw string) Result {
	if user.TotpEnabled != 1 {
		if IsRequired() {
			return Result{Reason: "系统已强制启用 MFA，请先登录 Web 端在「个人中心 → MFA 设置」完成绑定"}
		}
		if CheckPassword(raw, user.Password) {
			return Result{OK: true}
		}
		return Result{Reason: "密码错误"}
	}

	// codeFormat 二次验证口令只可能是 6 位数字或恢复码，先按格式过滤再比对，
	// 避免把恢复码误当成密码前缀参与 bcrypt 校验
	codeFormat := func(code string) bool {
		if len(code) == totpCodeLen {
			_, err := strconv.Atoi(code)
			return err == nil
		}
		return len(NormalizeRecoveryCode(code)) == recoveryPlain
	}

	// 整体口令先于任何切分参与密码校验（密码恰好以 6 位数字结尾时切分候选也会命中 bcrypt，
	// 若先试切分会消耗一次性恢复码或误报动态码错误，而非提示正确格式）
	if CheckPassword(raw, user.Password) {
		return Result{Reason: "已启用 MFA，请输入「密码+6位动态码」或「密码+恢复码」"}
	}
	for _, c := range ConcatCandidates(raw) {
		if !codeFormat(c.Code) || !CheckPassword(c.Password, user.Password) {
			continue
		}
		if VerifyCode(user, c.Code) {
			return Result{OK: true}
		}
		return Result{Reason: "动态验证码错误或已过期"}
	}
	return Result{Reason: "密码错误"}
}
