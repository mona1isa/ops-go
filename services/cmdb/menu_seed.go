package cmdb

import (
	"log"

	"github.com/zhany/ops-go/models"
)

// cmdbMenuSeed 资产管理模块的菜单与权限种子。
// RequestUrl/RequestMethod 是 Casbin 策略来源，必须填写，否则非超管角色无法被授予接口权限。
type cmdbMenuSeed struct {
	Name          string
	ParentKey     string
	Key           string
	OrderNum      int
	Path          string
	Component     string
	Type          string
	Icon          string
	Perms         string
	RequestUrl    string
	RequestMethod string
}

var cmdbMenuSeeds = []cmdbMenuSeed{
	{Name: "资产管理", Key: "cmdb", OrderNum: 9, Path: "/cmdb", Component: "layout/routerView/parent", Type: "M", Icon: "ele-Files"},

	{Name: "资产台账", Key: "asset", ParentKey: "cmdb", OrderNum: 1, Path: "/cmdb/asset", Component: "cmdb/asset/index", Type: "C", Icon: "ele-Notebook", Perms: "cmdb:asset:list", RequestUrl: "/api/cmdb/asset/page", RequestMethod: "POST"},
	{Name: "新增资产", ParentKey: "asset", OrderNum: 1, Type: "F", Perms: "cmdb:asset:add", RequestUrl: "/api/cmdb/asset/add", RequestMethod: "POST"},
	{Name: "编辑资产", ParentKey: "asset", OrderNum: 2, Type: "F", Perms: "cmdb:asset:edit", RequestUrl: "/api/cmdb/asset/edit", RequestMethod: "POST"},
	{Name: "删除资产", ParentKey: "asset", OrderNum: 3, Type: "F", Perms: "cmdb:asset:delete", RequestUrl: "/api/cmdb/asset/delete", RequestMethod: "POST"},
	{Name: "资产详情", ParentKey: "asset", OrderNum: 4, Type: "F", Perms: "cmdb:asset:detail", RequestUrl: "/api/cmdb/asset/detail", RequestMethod: "POST"},
	{Name: "状态流转", ParentKey: "asset", OrderNum: 5, Type: "F", Perms: "cmdb:asset:transition", RequestUrl: "/api/cmdb/asset/transition", RequestMethod: "POST"},
	{Name: "导入纳管主机", ParentKey: "asset", OrderNum: 6, Type: "F", Perms: "cmdb:asset:import", RequestUrl: "/api/cmdb/asset/import", RequestMethod: "POST"},
	{Name: "资产变更记录", ParentKey: "asset", OrderNum: 7, Type: "F", Perms: "cmdb:asset:changelog", RequestUrl: "/api/cmdb/asset/change-log", RequestMethod: "POST"},
	{Name: "设置资产标签", ParentKey: "asset", OrderNum: 8, Type: "F", Perms: "cmdb:tag:set", RequestUrl: "/api/cmdb/tag/set", RequestMethod: "POST"},

	{Name: "资产拓扑", Key: "topology", ParentKey: "cmdb", OrderNum: 2, Path: "/cmdb/topology", Component: "cmdb/topology/index", Type: "C", Icon: "ele-Share", Perms: "cmdb:topology:view", RequestUrl: "/api/cmdb/relation/graph-all", RequestMethod: "POST"},
	{Name: "邻域拓扑", ParentKey: "topology", OrderNum: 1, Type: "F", Perms: "cmdb:relation:graph", RequestUrl: "/api/cmdb/relation/graph", RequestMethod: "POST"},
	{Name: "新增关联", ParentKey: "topology", OrderNum: 2, Type: "F", Perms: "cmdb:relation:add", RequestUrl: "/api/cmdb/relation/add", RequestMethod: "POST"},
	{Name: "删除关联", ParentKey: "topology", OrderNum: 3, Type: "F", Perms: "cmdb:relation:delete", RequestUrl: "/api/cmdb/relation/delete", RequestMethod: "POST"},
	{Name: "资产关联列表", ParentKey: "topology", OrderNum: 4, Type: "F", Perms: "cmdb:relation:list", RequestUrl: "/api/cmdb/relation/list", RequestMethod: "POST"},
	{Name: "资产下拉选项", ParentKey: "topology", OrderNum: 5, Type: "F", Perms: "cmdb:asset:options", RequestUrl: "/api/cmdb/asset/options", RequestMethod: "POST"},

	{Name: "标签管理", Key: "tag", ParentKey: "cmdb", OrderNum: 3, Path: "/cmdb/tag", Component: "cmdb/tag/index", Type: "C", Icon: "ele-PriceTag", Perms: "cmdb:tag:list", RequestUrl: "/api/cmdb/tag/list", RequestMethod: "POST"},
	{Name: "新增标签", ParentKey: "tag", OrderNum: 1, Type: "F", Perms: "cmdb:tag:add", RequestUrl: "/api/cmdb/tag/add", RequestMethod: "POST"},
	{Name: "编辑标签", ParentKey: "tag", OrderNum: 2, Type: "F", Perms: "cmdb:tag:edit", RequestUrl: "/api/cmdb/tag/edit", RequestMethod: "POST"},
	{Name: "删除标签", ParentKey: "tag", OrderNum: 3, Type: "F", Perms: "cmdb:tag:delete", RequestUrl: "/api/cmdb/tag/delete", RequestMethod: "POST"},

	{Name: "IP管理", Key: "ipam", ParentKey: "cmdb", OrderNum: 4, Path: "/cmdb/ipam", Component: "cmdb/ipam/index", Type: "C", Icon: "ele-LocationInformation", Perms: "cmdb:subnet:list", RequestUrl: "/api/cmdb/subnet/page", RequestMethod: "POST"},
	{Name: "新增网段", ParentKey: "ipam", OrderNum: 1, Type: "F", Perms: "cmdb:subnet:add", RequestUrl: "/api/cmdb/subnet/add", RequestMethod: "POST"},
	{Name: "编辑网段", ParentKey: "ipam", OrderNum: 2, Type: "F", Perms: "cmdb:subnet:edit", RequestUrl: "/api/cmdb/subnet/edit", RequestMethod: "POST"},
	{Name: "删除网段", ParentKey: "ipam", OrderNum: 3, Type: "F", Perms: "cmdb:subnet:delete", RequestUrl: "/api/cmdb/subnet/delete", RequestMethod: "POST"},
	{Name: "网段使用情况", ParentKey: "ipam", OrderNum: 4, Type: "F", Perms: "cmdb:subnet:usage", RequestUrl: "/api/cmdb/subnet/usage", RequestMethod: "POST"},

	{Name: "证书到期", Key: "certAlert", ParentKey: "cmdb", OrderNum: 5, Path: "/cmdb/certAlert", Component: "cmdb/certAlert/index", Type: "C", Icon: "ele-Ticket", Perms: "cmdb:cert:view", RequestUrl: "/api/cmdb/cert/expiring", RequestMethod: "POST"},
	{Name: "证书告警规则列表", ParentKey: "certAlert", OrderNum: 1, Type: "F", Perms: "cmdb:cert:rule:list", RequestUrl: "/api/cmdb/cert/rule/page", RequestMethod: "POST"},
	{Name: "新增证书告警规则", ParentKey: "certAlert", OrderNum: 2, Type: "F", Perms: "cmdb:cert:rule:add", RequestUrl: "/api/cmdb/cert/rule/add", RequestMethod: "POST"},
	{Name: "编辑证书告警规则", ParentKey: "certAlert", OrderNum: 3, Type: "F", Perms: "cmdb:cert:rule:edit", RequestUrl: "/api/cmdb/cert/rule/edit", RequestMethod: "POST"},
	{Name: "删除证书告警规则", ParentKey: "certAlert", OrderNum: 4, Type: "F", Perms: "cmdb:cert:rule:delete", RequestUrl: "/api/cmdb/cert/rule/delete", RequestMethod: "POST"},
	{Name: "启停证书告警规则", ParentKey: "certAlert", OrderNum: 5, Type: "F", Perms: "cmdb:cert:rule:toggle", RequestUrl: "/api/cmdb/cert/rule/toggle", RequestMethod: "POST"},
	{Name: "静默证书告警规则", ParentKey: "certAlert", OrderNum: 6, Type: "F", Perms: "cmdb:cert:rule:silence", RequestUrl: "/api/cmdb/cert/rule/silence", RequestMethod: "POST"},
	{Name: "证书告警事件", ParentKey: "certAlert", OrderNum: 7, Type: "F", Perms: "cmdb:cert:events", RequestUrl: "/api/cmdb/cert/events", RequestMethod: "POST"},
}

// InitCmdbMenu 初始化资产管理菜单并授权给管理员角色。
// 逐条按名称检查，已存在的跳过、缺失的补齐，重复调用安全。
func InitCmdbMenu() {
	idByKey := map[string]int{}
	createdAny := false
	for _, seed := range cmdbMenuSeeds {
		var menu models.SysMenu
		if err := models.DB.Where("name = ? AND del_flag = ?", seed.Name, "0").First(&menu).Error; err == nil {
			key := seed.Key
			if key == "" {
				key = seed.Name
			}
			idByKey[key] = menu.ID
			continue
		}

		parentId := 0
		if seed.ParentKey != "" {
			parentId = idByKey[seed.ParentKey]
			if parentId == 0 {
				log.Printf("跳过菜单初始化 [%s]：父菜单 %s 未创建", seed.Name, seed.ParentKey)
				continue
			}
		}

		menu = models.SysMenu{
			Name:          seed.Name,
			ParentId:      parentId,
			OrderNum:      seed.OrderNum,
			Path:          seed.Path,
			Component:     seed.Component,
			Type:          seed.Type,
			Status:        true,
			KeepAlive:     seed.Type == "C",
			Icon:          seed.Icon,
			Perms:         seed.Perms,
			RequestUrl:    seed.RequestUrl,
			RequestMethod: seed.RequestMethod,
		}
		if err := models.DB.Create(&menu).Error; err != nil {
			log.Printf("初始化资产菜单失败 [%s]: %v", seed.Name, err)
			continue
		}
		if menu.ID == 0 {
			var existing models.SysMenu
			if err := models.DB.Where("name = ? AND del_flag = ?", seed.Name, "0").First(&existing).Error; err == nil {
				menu.ID = existing.ID
			}
		}
		createdAny = true
		key := seed.Key
		if key == "" {
			key = seed.Name
		}
		idByKey[key] = menu.ID
	}

	if createdAny {
		grantCmdbMenuToAdmin(idByKey)
		log.Println("资产管理菜单初始化完成")
	}
}

// grantCmdbMenuToAdmin 把新建菜单授予管理员角色，使角色权限界面可以看到并勾选
func grantCmdbMenuToAdmin(menuIds map[string]int) {
	var adminRole models.SysRole
	if err := models.DB.Where("id = ?", 1).First(&adminRole).Error; err != nil {
		log.Println("未找到管理员角色，跳过资产菜单授权")
		return
	}
	for key, menuId := range menuIds {
		if menuId == 0 {
			continue
		}
		var count int64
		if err := models.DB.Model(&models.SysRoleMenu{}).
			Where("role_id = ? AND menu_id = ?", adminRole.ID, menuId).Count(&count).Error; err != nil {
			log.Printf("检查资产菜单授权失败 [%s]: %v", key, err)
			continue
		}
		if count > 0 {
			continue
		}
		if err := models.DB.Create(&models.SysRoleMenu{RoleId: adminRole.ID, MenuId: menuId}).Error; err != nil {
			log.Printf("资产菜单授权失败 [%s]: %v", key, err)
		}
	}
}
