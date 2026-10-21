// 一次性授权脚本：把日志中心菜单授予指定角色，并按 sys_role_menu 重建 Casbin 策略。
// 用法（ops-go 目录下，读取 .env）：go run ./cmd/grantmenu -role=2
// 幂等：已有授权跳过，只补写缺失的关联与策略。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/models"
	systemService "github.com/zhany/ops-go/services/system"
)

var _ = config.RedisClient

func main() {
	roleId := flag.Int("role", 0, "要授权的角色 ID")
	flag.Parse()
	if *roleId <= 0 {
		fmt.Println("必须指定 -role=<角色ID>")
		os.Exit(1)
	}

	var role models.SysRole
	if err := models.DB.First(&role, *roleId).Error; err != nil {
		fmt.Printf("未找到角色 %d: %v\n", *roleId, err)
		os.Exit(1)
	}

	// 日志中心整棵菜单树：按 path 前缀取页面，再取其下全部按钮，另加告警规则弹窗依赖的跨模块按钮
	var moduleIds []int
	models.DB.Model(&models.SysMenu{}).
		Joins("LEFT JOIN sys_menu p ON p.id = sys_menu.parent_id").
		Where("sys_menu.del_flag = ? AND (sys_menu.path LIKE ? OR p.path LIKE ?)", "0", "/logs%", "/logs%").
		Pluck("sys_menu.id", &moduleIds)

	var crossIds []int
	models.DB.Model(&models.SysMenu{}).
		Where("del_flag = ? AND request_url = ?", "0", "/api/monitor/channel/options").
		Pluck("id", &crossIds)

	menuIds := append(moduleIds, crossIds...)
	var added, skipped int
	for _, menuId := range menuIds {
		var count int64
		models.DB.Model(&models.SysRoleMenu{}).Where("role_id = ? AND menu_id = ?", role.ID, menuId).Count(&count)
		if count > 0 {
			skipped++
			continue
		}
		if err := models.DB.Create(&models.SysRoleMenu{RoleId: role.ID, MenuId: menuId}).Error; err != nil {
			fmt.Printf("授权失败 menu_id=%d: %v\n", menuId, err)
			continue
		}
		added++
	}
	fmt.Printf("角色 %d(%s) 菜单授权：新增 %d 条，已存在 %d 条\n", role.ID, role.Name, added, skipped)

	systemService.SyncRolePolicies()
	fmt.Println("Casbin 策略同步完成")
}
