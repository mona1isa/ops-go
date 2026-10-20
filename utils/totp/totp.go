// Package totp 实现 RFC 6238 TOTP 动态口令（兼容 Google Authenticator / FreeOTP），
// 仅使用标准库，参数固定为业界默认：SHA1、30 秒步长、6 位数字。
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	period    = 30                                 // 时间步长（秒）
	digits    = 6                                  // 用户侧口令位数
	secretLen = 20                                 // 随机密钥字节数，base32 后 32 字符
	skewSteps = 1                                  // 允许的时钟偏移步数（±30s）
	codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 恢复码字符集，去掉易混的 I/O/0/1
)

// GenerateSecret 生成 base32 编码的随机密钥（不含填充符）
func GenerateSecret() (string, error) {
	buf := make([]byte, secretLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成 TOTP 密钥失败: %w", err)
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "="), nil
}

// hotp 计算 RFC 4226 的 HOTP 值，digits 指定位数
func hotp(key []byte, counter uint64, numDigits int) (string, error) {
	mac := hmac.New(sha1.New, key)
	buf := []byte{
		byte(counter >> 56), byte(counter >> 48), byte(counter >> 40), byte(counter >> 32),
		byte(counter >> 24), byte(counter >> 16), byte(counter >> 8), byte(counter),
	}
	if _, err := mac.Write(buf); err != nil {
		return "", err
	}
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0F
	value := (uint32(sum[offset])&0x7F)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	mod := uint32(1)
	for i := 0; i < numDigits; i++ {
		mod *= 10
	}
	code := value % mod
	return fmt.Sprintf("%0*d", numDigits, code), nil
}

// code 计算指定时刻的 TOTP（digits 位），供 Now 与测试向量共用
func code(base32Secret string, at time.Time, numDigits int) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(base32Secret))
	if err != nil {
		return "", fmt.Errorf("TOTP 密钥格式非法: %w", err)
	}
	return hotp(key, uint64(at.Unix()/int64(period)), numDigits)
}

// Now 返回当前时刻的 6 位口令
func Now(base32Secret string, at time.Time) (string, error) {
	return code(base32Secret, at, digits)
}

// Verify 校验用户输入的口令，允许 ±skewSteps 个时间步的时钟偏差；常量时间比较
func Verify(base32Secret, input string, at time.Time) bool {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(base32Secret))
	if err != nil {
		return false
	}
	input = strings.TrimSpace(input)
	if len(input) != digits {
		return false
	}
	counter := uint64(at.Unix() / int64(period))
	for i := -skewSteps; i <= skewSteps; i++ {
		c := counter + uint64(int64(i))
		expect, err := hotp(key, c, digits)
		if err != nil {
			return false
		}
		if hmac.Equal([]byte(expect), []byte(input)) {
			return true
		}
	}
	return false
}

// ProvisioningURL 生成 authenticator 扫码用的 otpauth URI
func ProvisioningURL(base32Secret, account, issuer string) string {
	label := url.PathEscape(fmt.Sprintf("%s:%s", issuer, account))
	q := url.Values{}
	q.Set("secret", strings.ToUpper(base32Secret))
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(digits))
	q.Set("period", fmt.Sprint(period))
	return fmt.Sprintf("otpauth://totp/%s?%s", label, q.Encode())
}

// GenerateRecoveryCodes 生成 n 个一次性恢复码，格式 XXXXXXXX-XXXXXXXX
func GenerateRecoveryCodes(n int) ([]string, error) {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return nil, fmt.Errorf("生成恢复码失败: %w", err)
		}
		groups := []string{"", ""}
		for j, b := range buf {
			groups[j/8] += string(codeChars[int(b)%len(codeChars)])
		}
		out = append(out, groups[0]+"-"+groups[1])
	}
	return out, nil
}
