package models

import (
	"testing"
)

// TestMakeRoleName 测试角色名称生成
func TestMakeRoleName(t *testing.T) {
	tests := []struct {
		roleId   int
		expected string
	}{
		{1, "role_1"},
		{100, "role_100"},
		{0, "role_0"},
		{-1, "role_-1"},
	}
	
	// 创建一个临时的 handler 来测试
	handler := &CasbinHandler{}
	
	for _, test := range tests {
		result := handler.MakeRoleName(test.roleId)
		if result != test.expected {
			t.Errorf("MakeRoleName(%d) = %s, expected %s", test.roleId, result, test.expected)
		}
	}
}

// TestAdminUserId 测试管理员 ID 常量
func TestAdminUserId(t *testing.T) {
	if AdminUserId != 1 {
		t.Errorf("AdminUserId = %d, expected 1", AdminUserId)
	}
}
