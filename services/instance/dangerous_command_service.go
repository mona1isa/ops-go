package instance

import (
	"errors"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// DangerousCommandRule 高危指令规则（内存缓存结构）
type DangerousCommandRule struct {
	ID          int
	Name        string
	Pattern     string
	MatchType   int8
	Description string
	Regex       *regexp.Regexp // 预编译的正则表达式
}

var (
	// dangerousCommandRules 内存缓存的高危指令规则
	dangerousCommandRules []*DangerousCommandRule
	// rulesMutex 读写锁保护缓存
	rulesMutex sync.RWMutex
	// rulesOnce 确保初始化只执行一次
	rulesOnce sync.Once
)

// InitDangerousCommands 初始化高危指令规则（从数据库加载到内存）
func InitDangerousCommands() {
	rulesOnce.Do(func() {
		InitDangerousCommandMenu()
		ReloadDangerousCommands()
		InitBuiltinDangerousCommands()
	})
}

// ReloadDangerousCommands 重新从数据库加载规则到内存缓存
func ReloadDangerousCommands() {
	var commands []models.OpsDangerousCommand
	if err := models.DB.Where("is_enabled = ? AND del_flag = ?", 1, 0).Find(&commands).Error; err != nil {
		log.Printf("加载高危指令规则失败: %v", err)
		return
	}

	newRules := make([]*DangerousCommandRule, 0, len(commands))
	for _, cmd := range commands {
		rule := &DangerousCommandRule{
			ID:          cmd.ID,
			Name:        cmd.Name,
			Pattern:     cmd.Pattern,
			MatchType:   cmd.MatchType,
			Description: cmd.Description,
		}
		// 正则匹配类型需要预编译
		if cmd.MatchType == models.MatchTypeRegex {
			if re, err := regexp.Compile(cmd.Pattern); err == nil {
				rule.Regex = re
			} else {
				log.Printf("编译正则表达式失败 [%s]: %v", cmd.Pattern, err)
			}
		}
		newRules = append(newRules, rule)
	}

	rulesMutex.Lock()
	dangerousCommandRules = newRules
	rulesMutex.Unlock()

	log.Printf("高危指令规则加载完成，共 %d 条", len(newRules))
}

// CheckCommand 检查命令是否命中高危指令规则
// isAdmin: 是否为管理员（管理员直接放行）
// 返回: blocked(是否被拦截), ruleName(规则名称), description(规则描述)
func CheckCommand(cmd string, isAdmin bool) (bool, string, string) {
	if isAdmin {
		return false, "", ""
	}

	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false, "", ""
	}

	rulesMutex.RLock()
	rules := dangerousCommandRules
	rulesMutex.RUnlock()

	for _, rule := range rules {
		if rule == nil {
			continue
		}
		switch rule.MatchType {
		case models.MatchTypeExact:
			if cmd == rule.Pattern {
				return true, rule.Name, rule.Description
			}
		case models.MatchTypePrefix:
			if strings.HasPrefix(cmd, rule.Pattern) {
				return true, rule.Name, rule.Description
			}
		case models.MatchTypeRegex:
			if rule.Regex != nil && rule.Regex.MatchString(cmd) {
				return true, rule.Name, rule.Description
			}
		}
	}

	return false, "", ""
}

// DangerousCommandService 高危指令规则服务
type DangerousCommandService struct{}

// List 分页查询高危指令规则
func (s *DangerousCommandService) List(pageNum, pageSize int, name string) (models.PageResult[models.OpsDangerousCommand], error) {
	return models.Paginate[models.OpsDangerousCommand](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		if name != "" {
			db = db.Where("name like ?", "%"+name+"%")
		}
		return db.Where("del_flag = ?", 0).Order("is_builtin desc, id desc")
	})
}

// Add 新增高危指令规则
func (s *DangerousCommandService) Add(command *models.OpsDangerousCommand) error {
	if command.Name == "" || command.Pattern == "" {
		return errors.New("规则名称和匹配模式不能为空")
	}
	if err := models.DB.Create(command).Error; err != nil {
		log.Printf("新增高危指令规则失败: %v", err)
		return errors.New("新增失败")
	}
	ReloadDangerousCommands()
	return nil
}

// Edit 编辑高危指令规则
func (s *DangerousCommandService) Edit(command *models.OpsDangerousCommand) error {
	if command.ID == 0 {
		return errors.New("规则ID不能为空")
	}
	if command.Name == "" || command.Pattern == "" {
		return errors.New("规则名称和匹配模式不能为空")
	}

	var existing models.OpsDangerousCommand
	if err := models.DB.First(&existing, command.ID).Error; err != nil {
		return errors.New("规则不存在")
	}

	// 内置规则只允许修改启用状态
	if existing.IsBuiltin == 1 {
		if err := models.DB.Model(&existing).Update("is_enabled", command.IsEnabled).Error; err != nil {
			return errors.New("更新失败")
		}
	} else {
		updates := map[string]interface{}{
			"name":        command.Name,
			"pattern":     command.Pattern,
			"match_type":  command.MatchType,
			"description": command.Description,
			"is_enabled":  command.IsEnabled,
		}
		if err := models.DB.Model(&existing).Updates(updates).Error; err != nil {
			return errors.New("更新失败")
		}
	}

	ReloadDangerousCommands()
	return nil
}

// Delete 删除高危指令规则
func (s *DangerousCommandService) Delete(id int) error {
	var existing models.OpsDangerousCommand
	if err := models.DB.First(&existing, id).Error; err != nil {
		return errors.New("规则不存在")
	}
	if existing.IsBuiltin == 1 {
		return errors.New("内置规则不允许删除")
	}
	if err := models.DB.Model(&existing).Update("del_flag", 1).Error; err != nil {
		return errors.New("删除失败")
	}
	ReloadDangerousCommands()
	return nil
}

// ToggleStatus 切换规则启用状态
func (s *DangerousCommandService) ToggleStatus(id int, isEnabled int8) error {
	if err := models.DB.Model(&models.OpsDangerousCommand{}).Where("id = ?", id).Update("is_enabled", isEnabled).Error; err != nil {
		return errors.New("状态更新失败")
	}
	ReloadDangerousCommands()
	return nil
}

// InitDangerousCommandMenu 初始化高危指令管理菜单
func InitDangerousCommandMenu() {
	var existing models.SysMenu
	if err := models.DB.Where("name = ? AND del_flag = ?", "高危指令管理", "0").First(&existing).Error; err == nil {
		return // 已存在，不重复初始化
	}

	// 查找任意一个 type='M' 的顶层菜单作为父级
	var parentMenu models.SysMenu
	if err := models.DB.Where("type = ? AND del_flag = ? AND status = ?", "M", "0", "1").Order("id asc").First(&parentMenu).Error; err != nil {
		log.Println("未找到任何菜单目录，跳过初始化高危指令菜单")
		return
	}

	menu := models.SysMenu{
		Name:       "高危指令管理",
		ParentId:   parentMenu.ID,
		OrderNum:   999,
		Path:       "/system/dangerousCommand",
		Component:  "dangerousCommand/index",
		Type:       "C",
		Status:     true,
		Icon:       "ele-WarningFilled",
		KeepAlive:  true,
		Perms:      "system:dangerousCommand:list",
	}
	if err := models.DB.Create(&menu).Error; err != nil {
		log.Printf("初始化高危指令管理菜单失败: %v", err)
		return
	}
	log.Printf("高危指令管理菜单初始化完成，父菜单: %s(ID:%d)", parentMenu.Name, parentMenu.ID)
}

// InitBuiltinDangerousCommands 初始化内置高危指令规则
func InitBuiltinDangerousCommands() {
	var count int64
	models.DB.Model(&models.OpsDangerousCommand{}).Where("is_builtin = ?", 1).Count(&count)
	if count > 0 {
		return // 已存在内置规则，不重复初始化
	}

	builtinCommands := []models.OpsDangerousCommand{
		{Name: "删除根目录", Pattern: `rm\s+-rf\s+/`, MatchType: models.MatchTypeRegex, Description: "禁止执行删除根目录及其子目录的操作", IsEnabled: 1, IsBuiltin: 1},
		{Name: "删除根目录通配", Pattern: `rm\s+-rf\s+/\*`, MatchType: models.MatchTypeRegex, Description: "禁止执行 rm -rf /* 操作", IsEnabled: 1, IsBuiltin: 1},
		{Name: "格式化文件系统", Pattern: `mkfs\.`, MatchType: models.MatchTypeRegex, Description: "禁止格式化文件系统", IsEnabled: 1, IsBuiltin: 1},
		{Name: "磁盘覆盖", Pattern: `dd\s+if=.*\s+of=/dev/[sh]d`, MatchType: models.MatchTypeRegex, Description: "禁止执行磁盘覆盖操作", IsEnabled: 1, IsBuiltin: 1},
		{Name: "写入磁盘设备", Pattern: `>\s*/dev/[sh]d`, MatchType: models.MatchTypeRegex, Description: "禁止向磁盘设备写入数据", IsEnabled: 1, IsBuiltin: 1},
		{Name: "Fork炸弹", Pattern: `:.*\(\)\s*\{.*:\|:.*\};.*:`, MatchType: models.MatchTypeRegex, Description: "禁止执行 fork 炸弹", IsEnabled: 1, IsBuiltin: 1},
		{Name: "系统关机", Pattern: `shutdown`, MatchType: models.MatchTypePrefix, Description: "禁止执行系统关机命令", IsEnabled: 1, IsBuiltin: 1},
		{Name: "系统重启", Pattern: `reboot`, MatchType: models.MatchTypePrefix, Description: "禁止执行系统重启命令", IsEnabled: 1, IsBuiltin: 1},
		{Name: "停止系统", Pattern: `halt`, MatchType: models.MatchTypeExact, Description: "禁止执行停止系统命令", IsEnabled: 1, IsBuiltin: 1},
		{Name: "关闭电源", Pattern: `poweroff`, MatchType: models.MatchTypePrefix, Description: "禁止执行关闭电源命令", IsEnabled: 1, IsBuiltin: 1},
		{Name: "切换运行级别0", Pattern: `init\s+0`, MatchType: models.MatchTypeRegex, Description: "禁止切换到关机运行级别", IsEnabled: 1, IsBuiltin: 1},
		{Name: "移动根目录到黑洞", Pattern: `mv\s+/\*\s+/dev/null`, MatchType: models.MatchTypeRegex, Description: "禁止将根目录移动到/dev/null", IsEnabled: 1, IsBuiltin: 1},
		{Name: "递归修改根目录权限", Pattern: `chmod\s+-R\s+000\s+/`, MatchType: models.MatchTypeRegex, Description: "禁止递归修改根目录权限为000", IsEnabled: 1, IsBuiltin: 1},
		{Name: "系统关机(systemctl)", Pattern: `systemctl\s+(poweroff|halt|reboot|shutdown)`, MatchType: models.MatchTypeRegex, Description: "禁止通过 systemctl 执行关机/重启操作", IsEnabled: 1, IsBuiltin: 1},
	}

	for _, cmd := range builtinCommands {
		if err := models.DB.Create(&cmd).Error; err != nil {
			log.Printf("初始化内置高危指令规则失败 [%s]: %v", cmd.Name, err)
		}
	}

	ReloadDangerousCommands()
	log.Println("内置高危指令规则初始化完成")
}
