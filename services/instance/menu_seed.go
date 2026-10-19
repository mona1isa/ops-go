package instance

import (
	"log"

	"github.com/zhany/ops-go/models"
)

// instanceImportMenuSeed 主机批量导入相关的按钮权限种子。
// RequestUrl/RequestMethod 是 Casbin 策略来源，必须填写，否则非超管角色无法被授予接口权限。
type instanceImportMenuSeed struct {
	Name          string
	OrderNum      int
	Perms         string
	RequestUrl    string
	RequestMethod string
}

var instanceImportMenuSeeds = []instanceImportMenuSeed{
	{Name: "批量导入主机", OrderNum: 20, Perms: "instance:import", RequestUrl: "/api/instance/import", RequestMethod: "POST"},
	{Name: "下载导入模板", OrderNum: 21, Perms: "instance:import:template", RequestUrl: "/api/instance/import/template", RequestMethod: "POST"},
}

// InitInstanceImportMenu 在「主机管理」菜单下补齐批量导入按钮权限，并授权给管理员角色。
// 逐条按名称检查，已存在的跳过、缺失的补齐，重复调用安全。
func InitInstanceImportMenu() {
	parentId := findInstanceMenuId()
	if parentId == 0 {
		log.Println("未找到主机管理菜单，跳过批量导入按钮初始化")
		return
	}

	var adminRole models.SysRole
	hasAdmin := models.DB.Where("id = ?", 1).First(&adminRole).Error == nil
	if !hasAdmin {
		log.Println("未找到管理员角色，跳过批量导入按钮授权")
	}

	for _, seed := range instanceImportMenuSeeds {
		var menu models.SysMenu
		if err := models.DB.Where("name = ? AND del_flag = ?", seed.Name, "0").First(&menu).Error; err == nil {
			continue
		}

		menu = models.SysMenu{
			Name:          seed.Name,
			ParentId:      parentId,
			OrderNum:      seed.OrderNum,
			Type:          "F",
			Status:        true,
			Perms:         seed.Perms,
			RequestUrl:    seed.RequestUrl,
			RequestMethod: seed.RequestMethod,
		}
		if err := models.DB.Create(&menu).Error; err != nil {
			log.Printf("初始化主机导入菜单失败 [%s]: %v", seed.Name, err)
			continue
		}
		if !hasAdmin || menu.ID == 0 {
			continue
		}
		if err := models.DB.Create(&models.SysRoleMenu{RoleId: adminRole.ID, MenuId: menu.ID}).Error; err != nil {
			log.Printf("主机导入菜单授权失败 [%s]: %v", seed.Name, err)
		}
	}
}

// findInstanceMenuId 定位主机管理页面菜单：优先按接口地址匹配，退回按常见菜单名匹配
func findInstanceMenuId() int {
	var menu models.SysMenu
	if err := models.DB.Where("request_url = ? AND type = ? AND del_flag = ?", "/api/instance/page", "C", "0").
		First(&menu).Error; err == nil {
		return menu.ID
	}
	for _, name := range []string{"主机管理", "主机列表", "实例管理"} {
		if err := models.DB.Where("name = ? AND del_flag = ?", name, "0").First(&menu).Error; err == nil {
			return menu.ID
		}
	}
	return 0
}
