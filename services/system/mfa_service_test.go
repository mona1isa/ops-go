package system

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zhany/ops-go/utils/totp"
	"golang.org/x/crypto/bcrypt"
)

func bcryptEqual(t *testing.T, hash, input string) bool {
	t.Helper()
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(input)) == nil
}

func TestRecoveryCodeHashRoundTrip(t *testing.T) {
	codes, err := totp.GenerateRecoveryCodes(3)
	if err != nil {
		t.Fatal(err)
	}
	hashes, err := hashRecoveryCodes(codes)
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 3 {
		t.Fatalf("期望 3 条哈希得到 %d", len(hashes))
	}
	encoded, _ := json.Marshal(hashes)
	parsed, err := parseRecoveryHashes(string(encoded))
	if err != nil || len(parsed) != 3 {
		t.Fatalf("哈希列表序列化往返失败: %v", err)
	}

	// 带格式差异（小写、保留连字符）的输入应能匹配同一哈希
	for _, c := range codes {
		varied := strings.ToLower(c)
		matched := false
		for _, h := range hashes {
			if bcryptEqual(t, h, normalizeRecoveryCode(varied)) {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("恢复码 %s 归一化后应命中哈希", c)
		}
	}
	// 错误输入不应命中任何哈希
	for _, h := range hashes {
		if bcryptEqual(t, h, normalizeRecoveryCode("XXXXXXXX-XXXXXXXX")) {
			t.Fatal("错误恢复码不应命中")
		}
	}
}

func TestParseRecoveryHashesEmpty(t *testing.T) {
	hashes, err := parseRecoveryHashes("")
	if err != nil || len(hashes) != 0 {
		t.Fatalf("空存储应返回空列表: %v %v", hashes, err)
	}
	if _, err := parseRecoveryHashes("not-json"); err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}

func TestNormalizeRecoveryCode(t *testing.T) {
	if got := normalizeRecoveryCode(" abcd-efgh-ijkl "); got != "ABCDEFGHIJKL" {
		t.Fatalf("归一化结果错误: %s", got)
	}
}
