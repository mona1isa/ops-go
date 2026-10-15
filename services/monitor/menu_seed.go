package monitor

import (
	"log"

	"github.com/zhany/ops-go/models"
)

// monitorMenuSeed 监控告警模块的菜单与权限种子。
// RequestUrl/RequestMethod 是 Casbin 策略来源，必须填写，否则非超管角色无法被授予接口权限。
type monitorMenuSeed struct {
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

var monitorMenuSeeds = []monitorMenuSeed{
	{Name: "监控告警", Key: "monitor", OrderNum: 8, Path: "/monitor", Component: "layout/routerView/parent", Type: "M", Icon: "ele-DataLine"},

	{Name: "实时监控面板", Key: "dashboard", ParentKey: "monitor", OrderNum: 1, Path: "/monitor/dashboard", Component: "monitor/dashboard/index", Type: "C", Icon: "ele-Odometer", Perms: "monitor:dashboard:view", RequestUrl: "/api/monitor/metrics/instant", RequestMethod: "POST"},
	{Name: "监控数据源状态", ParentKey: "dashboard", OrderNum: 1, Type: "F", Perms: "monitor:dashboard:status", RequestUrl: "/api/monitor/metrics/status", RequestMethod: "POST"},
	{Name: "监控主机列表", ParentKey: "dashboard", OrderNum: 2, Type: "F", Perms: "monitor:dashboard:hosts", RequestUrl: "/api/monitor/metrics/hosts", RequestMethod: "POST"},
	{Name: "监控趋势查询", ParentKey: "dashboard", OrderNum: 3, Type: "F", Perms: "monitor:dashboard:range", RequestUrl: "/api/monitor/metrics/range", RequestMethod: "POST"},

	{Name: "采集目标", Key: "target", ParentKey: "monitor", OrderNum: 2, Path: "/monitor/target", Component: "monitor/target/index", Type: "C", Icon: "ele-Aim", Perms: "monitor:target:list", RequestUrl: "/api/monitor/target/page", RequestMethod: "POST"},
	{Name: "新增采集目标", ParentKey: "target", OrderNum: 1, Type: "F", Perms: "monitor:target:add", RequestUrl: "/api/monitor/target/add", RequestMethod: "POST"},
	{Name: "编辑采集目标", ParentKey: "target", OrderNum: 2, Type: "F", Perms: "monitor:target:edit", RequestUrl: "/api/monitor/target/edit", RequestMethod: "POST"},
	{Name: "删除采集目标", ParentKey: "target", OrderNum: 3, Type: "F", Perms: "monitor:target:delete", RequestUrl: "/api/monitor/target/delete", RequestMethod: "POST"},
	{Name: "启停采集目标", ParentKey: "target", OrderNum: 4, Type: "F", Perms: "monitor:target:toggle", RequestUrl: "/api/monitor/target/toggle", RequestMethod: "POST"},
	{Name: "同步采集目标", ParentKey: "target", OrderNum: 5, Type: "F", Perms: "monitor:target:sync", RequestUrl: "/api/monitor/target/sync", RequestMethod: "POST"},
	{Name: "手动采集", ParentKey: "target", OrderNum: 6, Type: "F", Perms: "monitor:target:collect", RequestUrl: "/api/monitor/target/collect", RequestMethod: "POST"},
	{Name: "查询主机绑定凭证", ParentKey: "target", OrderNum: 7, Type: "F", Perms: "monitor:target:keys", RequestUrl: "/api/monitor/target/instance-keys", RequestMethod: "POST"},

	{Name: "告警规则", Key: "rule", ParentKey: "monitor", OrderNum: 3, Path: "/monitor/rule", Component: "monitor/rule/index", Type: "C", Icon: "ele-AlarmClock", Perms: "monitor:rule:list", RequestUrl: "/api/monitor/rule/page", RequestMethod: "POST"},
	{Name: "新增告警规则", ParentKey: "rule", OrderNum: 1, Type: "F", Perms: "monitor:rule:add", RequestUrl: "/api/monitor/rule/add", RequestMethod: "POST"},
	{Name: "编辑告警规则", ParentKey: "rule", OrderNum: 2, Type: "F", Perms: "monitor:rule:edit", RequestUrl: "/api/monitor/rule/edit", RequestMethod: "POST"},
	{Name: "删除告警规则", ParentKey: "rule", OrderNum: 3, Type: "F", Perms: "monitor:rule:delete", RequestUrl: "/api/monitor/rule/delete", RequestMethod: "POST"},
	{Name: "启停告警规则", ParentKey: "rule", OrderNum: 4, Type: "F", Perms: "monitor:rule:toggle", RequestUrl: "/api/monitor/rule/toggle", RequestMethod: "POST"},
	{Name: "告警静默", ParentKey: "rule", OrderNum: 5, Type: "F", Perms: "monitor:rule:silence", RequestUrl: "/api/monitor/rule/silence", RequestMethod: "POST"},
	{Name: "规则试算", ParentKey: "rule", OrderNum: 6, Type: "F", Perms: "monitor:rule:test", RequestUrl: "/api/monitor/rule/test", RequestMethod: "POST"},

	{Name: "告警事件", Key: "event", ParentKey: "monitor", OrderNum: 4, Path: "/monitor/event", Component: "monitor/event/index", Type: "C", Icon: "ele-Warning", Perms: "monitor:event:list", RequestUrl: "/api/monitor/event/page", RequestMethod: "POST"},
	{Name: "告警事件详情", ParentKey: "event", OrderNum: 1, Type: "F", Perms: "monitor:event:detail", RequestUrl: "/api/monitor/event/detail", RequestMethod: "POST"},
	{Name: "处理告警事件", ParentKey: "event", OrderNum: 2, Type: "F", Perms: "monitor:event:ack", RequestUrl: "/api/monitor/event/ack", RequestMethod: "POST"},
	{Name: "删除告警事件", ParentKey: "event", OrderNum: 3, Type: "F", Perms: "monitor:event:delete", RequestUrl: "/api/monitor/event/delete", RequestMethod: "POST"},

	{Name: "通知渠道", Key: "channel", ParentKey: "monitor", OrderNum: 5, Path: "/monitor/channel", Component: "monitor/channel/index", Type: "C", Icon: "ele-Promotion", Perms: "monitor:channel:list", RequestUrl: "/api/monitor/channel/page", RequestMethod: "POST"},
	{Name: "渠道下拉列表", ParentKey: "channel", OrderNum: 1, Type: "F", Perms: "monitor:channel:options", RequestUrl: "/api/monitor/channel/options", RequestMethod: "POST"},
	{Name: "新增通知渠道", ParentKey: "channel", OrderNum: 2, Type: "F", Perms: "monitor:channel:add", RequestUrl: "/api/monitor/channel/add", RequestMethod: "POST"},
	{Name: "编辑通知渠道", ParentKey: "channel", OrderNum: 3, Type: "F", Perms: "monitor:channel:edit", RequestUrl: "/api/monitor/channel/edit", RequestMethod: "POST"},
	{Name: "删除通知渠道", ParentKey: "channel", OrderNum: 4, Type: "F", Perms: "monitor:channel:delete", RequestUrl: "/api/monitor/channel/delete", RequestMethod: "POST"},
	{Name: "启停通知渠道", ParentKey: "channel", OrderNum: 5, Type: "F", Perms: "monitor:channel:toggle", RequestUrl: "/api/monitor/channel/toggle", RequestMethod: "POST"},
	{Name: "渠道测试发送", ParentKey: "channel", OrderNum: 6, Type: "F", Perms: "monitor:channel:test", RequestUrl: "/api/monitor/channel/test", RequestMethod: "POST"},
}

// InitMonitorMenu 初始化监控告警菜单并授权给管理员角色。
// 逐条按名称检查，已存在的跳过、缺失的补齐，因此新增按钮在存量部署上也会生效，重复调用安全。
func InitMonitorMenu() {
	idByKey := map[string]int{}
	createdAny := false
	for _, seed := range monitorMenuSeeds {
		var menu models.SysMenu
		if err := models.DB.Where("name = ? AND del_flag = ?", seed.Name, "0").First(&menu).Error; err == nil {
			// 菜单已存在，仅记录 ID 供子级引用
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
			log.Printf("初始化监控菜单失败 [%s]: %v", seed.Name, err)
			continue
		}
		// sys_menu.name 上有唯一索引，并发创建时按已存在的记录取 ID
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
		grantMonitorMenuToAdmin(idByKey)
		log.Println("监控告警菜单初始化完成")
	}
}

// grantMonitorMenuToAdmin 把新建菜单授予管理员角色，使角色权限界面可以看到并勾选
func grantMonitorMenuToAdmin(menuIds map[string]int) {
	var adminRole models.SysRole
	if err := models.DB.Where("id = ?", 1).First(&adminRole).Error; err != nil {
		log.Println("未找到管理员角色，跳过监控菜单授权")
		return
	}
	for key, menuId := range menuIds {
		if menuId == 0 {
			continue
		}
		var count int64
		if err := models.DB.Model(&models.SysRoleMenu{}).
			Where("role_id = ? AND menu_id = ?", adminRole.ID, menuId).Count(&count).Error; err != nil {
			log.Printf("检查监控菜单授权失败 [%s]: %v", key, err)
			continue
		}
		if count > 0 {
			continue
		}
		if err := models.DB.Create(&models.SysRoleMenu{RoleId: adminRole.ID, MenuId: menuId}).Error; err != nil {
			log.Printf("监控菜单授权失败 [%s]: %v", key, err)
		}
	}
}
