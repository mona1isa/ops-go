package system

import (
	"testing"

	"github.com/zhany/ops-go/models"
)

func TestPoliciesOfMenusSkipsIncompleteAndDedupes(t *testing.T) {
	menus := []models.SysMenu{
		{RequestUrl: "/api/logs/alert/page", RequestMethod: "POST"},
		{RequestUrl: "/api/logs/alert/add", RequestMethod: "POST"},
		// 同一接口被页面菜单和按钮菜单重复引用时必须去重，否则批量写入报 MySQL 1869
		{RequestUrl: "/api/logs/alert/page", RequestMethod: "POST"},
		{RequestUrl: "", RequestMethod: "POST"},
		{RequestUrl: "/api/logs/alert/edit", RequestMethod: ""},
		{RequestUrl: "/api/logs/alert/page", RequestMethod: "GET"},
	}

	got := policiesOfMenus("role_2", menus)
	want := map[string]bool{
		"role_2\t/api/logs/alert/page\tPOST": true,
		"role_2\t/api/logs/alert/add\tPOST":   true,
		"role_2\t/api/logs/alert/page\tGET":   true,
	}
	if len(got) != len(want) {
		t.Fatalf("策略条数 = %d，期望 %d：%v", len(got), len(want), got)
	}
	for _, rule := range got {
		key := rule[0] + "\t" + rule[1] + "\t" + rule[2]
		if !want[key] {
			t.Errorf("意外策略 %s", key)
		}
	}
}

func TestSamePolicies(t *testing.T) {
	a := [][]string{{"role_2", "/api/a", "POST"}, {"role_2", "/api/b", "POST"}}
	b := [][]string{{"role_2", "/api/b", "POST"}, {"role_2", "/api/a", "POST"}}
	if !samePolicies(a, b) {
		t.Error("顺序不同应视为相同")
	}
	if samePolicies(a, [][]string{{"role_2", "/api/a", "POST"}}) {
		t.Error("条数不同应视为不同")
	}
	if samePolicies(a, [][]string{{"role_2", "/api/a", "POST"}, {"role_2", "/api/c", "POST"}}) {
		t.Error("内容不同应视为不同")
	}
	if !samePolicies(nil, nil) {
		t.Error("两者皆空应视为相同")
	}
}
