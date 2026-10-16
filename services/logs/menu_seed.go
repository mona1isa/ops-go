package logs

import (
	"log"

	"github.com/zhany/ops-go/models"
)

// logMenuSeed 日志中心模块的菜单与权限种子。
// RequestUrl/RequestMethod 是 Casbin 策略来源，必须填写且与路由完全一致，否则非超管角色无法被授予接口权限。
type logMenuSeed struct {
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

var logMenuSeeds = []logMenuSeed{
	{Name: "日志中心", Key: "logs", OrderNum: 9, Path: "/logs", Component: "layout/routerView/parent", Type: "M", Icon: "ele-Document"},

	{Name: "日志检索", Key: "query", ParentKey: "logs", OrderNum: 1, Path: "/logs/query", Component: "logs/query/index", Type: "C", Icon: "ele-Search", Perms: "logs:query:view", RequestUrl: "/api/logs/query", RequestMethod: "POST"},
	{Name: "日志时间分布", ParentKey: "query", OrderNum: 1, Type: "F", Perms: "logs:query:histogram", RequestUrl: "/api/logs/histogram", RequestMethod: "POST"},
	{Name: "日志上下文", ParentKey: "query", OrderNum: 2, Type: "F", Perms: "logs:query:context", RequestUrl: "/api/logs/context", RequestMethod: "POST"},
	{Name: "日志标签列表", ParentKey: "query", OrderNum: 3, Type: "F", Perms: "logs:query:labels", RequestUrl: "/api/logs/labels", RequestMethod: "POST"},
	{Name: "日志标签取值", ParentKey: "query", OrderNum: 4, Type: "F", Perms: "logs:query:labelValues", RequestUrl: "/api/logs/label-values", RequestMethod: "POST"},
	{Name: "日志数据源状态", ParentKey: "query", OrderNum: 5, Type: "F", Perms: "logs:query:status", RequestUrl: "/api/logs/status", RequestMethod: "POST"},

	{Name: "采集源", Key: "source", ParentKey: "logs", OrderNum: 2, Path: "/logs/source", Component: "logs/source/index", Type: "C", Icon: "ele-Files", Perms: "logs:source:list", RequestUrl: "/api/logs/source/page", RequestMethod: "POST"},
	{Name: "新增采集源", ParentKey: "source", OrderNum: 1, Type: "F", Perms: "logs:source:add", RequestUrl: "/api/logs/source/add", RequestMethod: "POST"},
	{Name: "编辑采集源", ParentKey: "source", OrderNum: 2, Type: "F", Perms: "logs:source:edit", RequestUrl: "/api/logs/source/edit", RequestMethod: "POST"},
	{Name: "删除采集源", ParentKey: "source", OrderNum: 3, Type: "F", Perms: "logs:source:delete", RequestUrl: "/api/logs/source/delete", RequestMethod: "POST"},
	{Name: "启停采集源", ParentKey: "source", OrderNum: 4, Type: "F", Perms: "logs:source:toggle", RequestUrl: "/api/logs/source/toggle", RequestMethod: "POST"},
	{Name: "手动拉取", ParentKey: "source", OrderNum: 5, Type: "F", Perms: "logs:source:collect", RequestUrl: "/api/logs/source/collect", RequestMethod: "POST"},
	{Name: "重置读取位点", ParentKey: "source", OrderNum: 6, Type: "F", Perms: "logs:source:reset", RequestUrl: "/api/logs/source/reset", RequestMethod: "POST"},
	{Name: "采集源下拉", ParentKey: "source", OrderNum: 7, Type: "F", Perms: "logs:source:options", RequestUrl: "/api/logs/source/options", RequestMethod: "POST"},
	{Name: "查询主机绑定凭证", ParentKey: "source", OrderNum: 8, Type: "F", Perms: "logs:source:keys", RequestUrl: "/api/logs/source/instance-keys", RequestMethod: "POST"},

	{Name: "日志告警", Key: "alert", ParentKey: "logs", OrderNum: 3, Path: "/logs/alert", Component: "logs/alert/index", Type: "C", Icon: "ele-Bell", Perms: "logs:alert:list", RequestUrl: "/api/logs/alert/page", RequestMethod: "POST"},
	{Name: "新增日志告警规则", ParentKey: "alert", OrderNum: 1, Type: "F", Perms: "logs:alert:add", RequestUrl: "/api/logs/alert/add", RequestMethod: "POST"},
	{Name: "编辑日志告警规则", ParentKey: "alert", OrderNum: 2, Type: "F", Perms: "logs:alert:edit", RequestUrl: "/api/logs/alert/edit", RequestMethod: "POST"},
	{Name: "删除日志告警规则", ParentKey: "alert", OrderNum: 3, Type: "F", Perms: "logs:alert:delete", RequestUrl: "/api/logs/alert/delete", RequestMethod: "POST"},
	{Name: "启停日志告警规则", ParentKey: "alert", OrderNum: 4, Type: "F", Perms: "logs:alert:toggle", RequestUrl: "/api/logs/alert/toggle", RequestMethod: "POST"},
	{Name: "日志告警静默", ParentKey: "alert", OrderNum: 5, Type: "F", Perms: "logs:alert:silence", RequestUrl: "/api/logs/alert/silence", RequestMethod: "POST"},
	{Name: "日志告警试算", ParentKey: "alert", OrderNum: 6, Type: "F", Perms: "logs:alert:test", RequestUrl: "/api/logs/alert/test", RequestMethod: "POST"},
	{Name: "日志告警事件", ParentKey: "alert", OrderNum: 7, Type: "F", Perms: "logs:alert:events", RequestUrl: "/api/logs/alert/events", RequestMethod: "POST"},

	{Name: "登录日志", Key: "login", ParentKey: "logs", OrderNum: 4, Path: "/logs/login", Component: "logs/login/index", Type: "C", Icon: "ele-Key", Perms: "logs:login:list", RequestUrl: "/api/logs/login/page", RequestMethod: "POST"},
	{Name: "登录行为分析", ParentKey: "login", OrderNum: 1, Type: "F", Perms: "logs:login:analyze", RequestUrl: "/api/logs/login/analyze", RequestMethod: "POST"},

	{Name: "操作审计", Key: "audit", ParentKey: "logs", OrderNum: 5, Path: "/logs/audit", Component: "logs/audit/index", Type: "C", Icon: "ele-Checked", Perms: "logs:audit:list", RequestUrl: "/api/logs/audit/page", RequestMethod: "POST"},
	{Name: "操作行为分析", ParentKey: "audit", OrderNum: 1, Type: "F", Perms: "logs:audit:analyze", RequestUrl: "/api/logs/audit/analyze", RequestMethod: "POST"},
}

// InitLogMenu 初始化日志中心菜单并授权给管理员角色。
// 逐条按名称检查，已存在的跳过、缺失的补齐，因此新增按钮在存量部署上也会生效，重复调用安全。
func InitLogMenu() {
	idByKey := map[string]int{}
	createdAny := false
	for _, seed := range logMenuSeeds {
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
				log.Printf("跳过日志菜单初始化 [%s]：父菜单 %s 未创建", seed.Name, seed.ParentKey)
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
			log.Printf("初始化日志菜单失败 [%s]: %v", seed.Name, err)
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
		grantLogMenuToAdmin(idByKey)
		log.Println("日志中心菜单初始化完成")
	}
}

// grantLogMenuToAdmin 把新建菜单授予管理员角色，使角色权限界面可以看到并勾选
func grantLogMenuToAdmin(menuIds map[string]int) {
	var adminRole models.SysRole
	if err := models.DB.Where("id = ?", 1).First(&adminRole).Error; err != nil {
		log.Println("未找到管理员角色，跳过日志菜单授权")
		return
	}
	for key, menuId := range menuIds {
		if menuId == 0 {
			continue
		}
		var count int64
		if err := models.DB.Model(&models.SysRoleMenu{}).
			Where("role_id = ? AND menu_id = ?", adminRole.ID, menuId).Count(&count).Error; err != nil {
			log.Printf("检查日志菜单授权失败 [%s]: %v", key, err)
			continue
		}
		if count > 0 {
			continue
		}
		if err := models.DB.Create(&models.SysRoleMenu{RoleId: adminRole.ID, MenuId: menuId}).Error; err != nil {
			log.Printf("日志菜单授权失败 [%s]: %v", key, err)
		}
	}
}
