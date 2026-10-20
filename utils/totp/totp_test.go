package totp

import (
	"encoding/base32"
	"regexp"
	"strings"
	"testing"
	"time"
)

// RFC 6238 附录 B 测试向量（密钥 "12345678901234567890"，SHA1，8 位口令）
const rfcSecretBase32 = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func TestRFC6238Vectors(t *testing.T) {
	vectors := []struct {
		unix int64
		code string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	}
	for _, v := range vectors {
		got, err := code(rfcSecretBase32, time.Unix(v.unix, 0), 8)
		if err != nil {
			t.Fatalf("时间 %d 计算失败: %v", v.unix, err)
		}
		if got != v.code {
			t.Errorf("时间 %d 期望 %s 得到 %s", v.unix, v.code, got)
		}
	}
}

func TestVerifyWindow(t *testing.T) {
	now := time.Unix(1111111111, 0) // 该时刻 6 位口令为 140504（8 位 14050471 的前 6 位）
	want, err := Now(rfcSecretBase32, now)
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(rfcSecretBase32, want, now) {
		t.Fatal("当前口令应通过")
	}
	// 前一步与后一步（±30s）仍在容忍窗口内
	if !Verify(rfcSecretBase32, want, now.Add(period*time.Second)) {
		t.Fatal("漂移 30 秒仍应通过")
	}
	if !Verify(rfcSecretBase32, want, now.Add(-period*time.Second)) {
		t.Fatal("漂移 -30 秒仍应通过")
	}
	// 超出两步窗口应拒绝
	if Verify(rfcSecretBase32, want, now.Add(3*period*time.Second)) {
		t.Fatal("漂移 90 秒应拒绝")
	}
	if Verify(rfcSecretBase32, "000000", now) {
		t.Fatal("错误口令应拒绝")
	}
	if Verify("!!invalid!!", want, now) {
		t.Fatal("非法密钥应拒绝")
	}
	if Verify(rfcSecretBase32, "12345", now) {
		t.Fatal("位数不足应拒绝")
	}
}

func TestGenerateSecret(t *testing.T) {
	s1, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	s2, _ := GenerateSecret()
	if s1 == s2 {
		t.Fatal("两次生成的密钥不应相同")
	}
	if _, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s1); err != nil {
		t.Fatalf("密钥应为合法 base32: %v", err)
	}
}

func TestProvisioningURL(t *testing.T) {
	uri := ProvisioningURL(rfcSecretBase32, "admin", "Ops-Go")
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Fatalf("前缀错误: %s", uri)
	}
	for _, want := range []string{"Ops-Go:admin", "secret=" + rfcSecretBase32, "issuer=Ops-Go", "digits=6", "period=30"} {
		if !strings.Contains(uri, want) {
			t.Errorf("URI 缺少 %s: %s", want, uri)
		}
	}
}

func TestRecoveryCodes(t *testing.T) {
	codes, err := GenerateRecoveryCodes(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 10 {
		t.Fatalf("期望 10 个得到 %d", len(codes))
	}
	pattern := regexp.MustCompile(`^[A-HJ-NP-Z2-9]{8}-[A-HJ-NP-Z2-9]{8}$`)
	seen := map[string]bool{}
	for _, c := range codes {
		if !pattern.MatchString(c) {
			t.Errorf("恢复码格式非法: %s", c)
		}
		if seen[c] {
			t.Errorf("恢复码重复: %s", c)
		}
		seen[c] = true
	}
}
