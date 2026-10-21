package mfa

import (
	"testing"

	"github.com/zhany/ops-go/models"
	"golang.org/x/crypto/bcrypt"
)

func hashed(t *testing.T, pass string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

func TestConcatCandidates(t *testing.T) {
	// 密码 + 6 位动态码（17/16 位切分不构成合法前缀或被格式过滤，最终命中 6 位切分）
	got := ConcatCandidates("Passw0rd!123456")
	if !containsCandidate(got, Candidate{Password: "Passw0rd!", Code: "123456"}) {
		t.Fatalf("6 位动态码切分缺失: %+v", got)
	}
	// 密码 + 带连字符恢复码：17 位切分应排在最前
	got = ConcatCandidates("pass1ABCDEFGH-EFGHJKLM")
	want := []Candidate{{"pass1", "ABCDEFGH-EFGHJKLM"}, {"pass1A", "BCDEFGH-EFGHJKLM"}}
	if len(got) < 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("恢复码切分顺序错误: %+v", got)
	}
	// 过短输入无法形成候选
	if len(ConcatCandidates("12345")) != 0 {
		t.Fatal("长度不足时不应有切分候选")
	}
	// 前缀为空不产生候选
	for _, c := range ConcatCandidates("ABCDEFGH-EFGHJKL2") {
		if c.Password == "" {
			t.Fatal("不应产生空前缀候选")
		}
	}
}

func containsCandidate(list []Candidate, want Candidate) bool {
	for _, c := range list {
		if c == want {
			return true
		}
	}
	return false
}

func TestAuthBastionPassword_NoMfa(t *testing.T) {
	t.Setenv("MFA_REQUIRED", "")
	user := &models.SysUser{Password: hashed(t, "secret123")}
	if r := AuthBastionPassword(user, "secret123"); !r.OK {
		t.Fatalf("未绑定 MFA 用户纯密码应通过: %+v", r)
	}
	if r := AuthBastionPassword(user, "wrong"); r.OK || r.Reason != "密码错误" {
		t.Fatalf("错误密码应被拒绝: %+v", r)
	}
}

func TestAuthBastionPassword_MfaRequiredUnbound(t *testing.T) {
	t.Setenv("MFA_REQUIRED", "true")
	user := &models.SysUser{Password: hashed(t, "secret123")}
	if r := AuthBastionPassword(user, "secret123"); r.OK {
		t.Fatal("强制 MFA 时未绑定用户不应通过")
	}
}

func TestAuthBastionPassword_ConcatTotp(t *testing.T) {
	t.Setenv("MFA_REQUIRED", "")
	// TotpSecret 非法（解密失败/空密钥）时动态码永远校验不过，用于验证失败分支的提示语
	user := &models.SysUser{Password: hashed(t, "secret123"), TotpEnabled: 1, TotpSecret: "!!!"}

	if r := AuthBastionPassword(user, "secret123"); r.OK ||
		r.Reason != "已启用 MFA，请输入「密码+6位动态码」或「密码+恢复码」" {
		t.Fatalf("缺少动态码应提示拼接格式: %+v", r)
	}
	if r := AuthBastionPassword(user, "secret123123456"); r.OK || r.Reason != "动态验证码错误或已过期" {
		t.Fatalf("动态码错误应明确提示: %+v", r)
	}
	if r := AuthBastionPassword(user, "wrongpass123456"); r.OK || r.Reason != "密码错误" {
		t.Fatalf("密码错误应提示密码错误: %+v", r)
	}
	if r := AuthBastionPassword(user, "secret12312345"); r.OK || r.Reason != "密码错误" {
		t.Fatalf("5 位数字后缀不构成合法切分，应提示密码错误: %+v", r)
	}
}

func TestVerifyTotpCodeInvalidSecret(t *testing.T) {
	user := &models.SysUser{TotpEnabled: 1, TotpSecret: "!!!"}
	if VerifyTotpCode(user, "123456") {
		t.Fatal("非法密钥不应校验通过")
	}
}
