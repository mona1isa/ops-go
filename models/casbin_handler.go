package models

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"sync"
	"time"
)

var (
	initOnce   sync.Once
	initError  error
	Casbin     *CasbinHandler
)

type CasbinHandler struct {
	enforcer *casbin.SyncedEnforcer
	mu       sync.RWMutex
}

// init 初始化 Casbin，使用 sync.Once 确保只执行一次
func (c *CasbinHandler) init() {
	initOnce.Do(func() {
		initError = c.doInit()
	})
}

// doInit 实际的初始化逻辑
func (c *CasbinHandler) doInit() error {
	adapter, err := gormadapter.NewAdapterByDB(DB)
	if err != nil {
		return fmt.Errorf("创建 Casbin adapter 失败: %w", err)
	}
	c.enforcer, err = casbin.NewSyncedEnforcer("config/casbin_rbac.conf", adapter)
	if err != nil {
		return fmt.Errorf("创建 Casbin enforcer 失败: %w", err)
	}
	// 配置自动刷新策略间隔（30秒）
	c.enforcer.StartAutoLoadPolicy(30 * time.Second)
	// 启用自动保存策略
	c.enforcer.EnableAutoSave(true)
	// 仅在开发环境启用日志
	// c.enforcer.EnableLog(true)
	// 加载策略
	if err = c.enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("加载 Casbin 策略失败: %w", err)
	}
	return nil
}

// IsInitialized 检查 Casbin 是否已初始化成功
func (c *CasbinHandler) IsInitialized() bool {
	c.init()
	return initError == nil && c.enforcer != nil
}

// GetInitError 获取初始化错误
func (c *CasbinHandler) GetInitError() error {
	c.init()
	return initError
}

// Enforcer Casbin权限验证
func (c *CasbinHandler) Enforcer(user, uri, method string) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enforcer.Enforce(user, uri, method)
}

// EnforceWithRoles 检查用户是否有权限（考虑角色状态）
// 返回: allowed-是否有权限, enabledRoles-启用的角色列表, error-错误
func (c *CasbinHandler) EnforceWithRoles(user, uri, method string, userRoles []SysUserRole) (bool, []int, error) {
	c.init()
	if initError != nil {
		return false, nil, initError
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 检查用户直接权限
	allowed, err := c.enforcer.Enforce(user, uri, method)
	if err != nil {
		return false, nil, err
	}
	if allowed {
		// 获取用户启用的角色ID列表
		var enabledRoleIds []int
		for _, ur := range userRoles {
			enabledRoleIds = append(enabledRoleIds, ur.RoleId)
		}
		return true, enabledRoleIds, nil
	}
	return false, nil, nil
}

// GetEnabledRolesForUser 获取用户启用的角色ID列表
func (c *CasbinHandler) GetEnabledRolesForUser(user string) ([]int, error) {
	c.init()
	if initError != nil {
		return nil, initError
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	roles, err := c.enforcer.GetRolesForUser(user)
	if err != nil {
		return nil, err
	}

	var roleIds []int
	for _, role := range roles {
		var roleId int
		if _, err := fmt.Sscanf(role, "role_%d", &roleId); err == nil {
			roleIds = append(roleIds, roleId)
		}
	}
	return roleIds, nil
}

// ReloadPolicy 重新加载策略（手动刷新）
func (c *CasbinHandler) ReloadPolicy() error {
	c.init()
	if initError != nil {
		return initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enforcer.LoadPolicy()
}

// AddPolicy 添加策略
func (c *CasbinHandler) AddPolicy(roleId int, uri, method string) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enforcer.AddPolicy(c.MakeRoleName(roleId), uri, method)
}

// MakeRoleName 拼接角色ID，为了防止角色与用户名冲突
func (c *CasbinHandler) MakeRoleName(roleId int) string {
	return fmt.Sprintf("role_%d", roleId)
}

// AddPolicies 添加策略
func (c *CasbinHandler) AddPolicies(rules [][]string) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enforcer.AddPolicies(rules)
}

// ClearAllPolicies 清除所有策略（慎用）
func (c *CasbinHandler) ClearAllPolicies() (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enforcer.RemoveFilteredPolicy(0)
}

// DeleteRolePolicy 删除角色下的权限
func (c *CasbinHandler) DeleteRolePolicy(roleId int) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enforcer.RemoveFilteredNamedPolicy("p", 0, c.MakeRoleName(roleId))
}

// DeleteRoleById 根据角色ID删除角色（包含p和g）
func (c *CasbinHandler) DeleteRoleById(roleId int) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enforcer.DeleteRole(c.MakeRoleName(roleId))
}

// DeleteRole 删除角色关联
func (c *CasbinHandler) DeleteRole(roleId int) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// 删除所有用户与该角色的关联（g 策略中 role 是第二个字段）
	return c.enforcer.RemoveFilteredNamedGroupingPolicy("g", 1, c.MakeRoleName(roleId))
}

// ClearUserRole 清除用户特定角色
func (c *CasbinHandler) ClearUserRole(roleId int, user string) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// 删除特定的用户-角色关联
	return c.enforcer.RemoveGroupingPolicy(user, c.MakeRoleName(roleId))
}

// HasRoleForUser 判断用户是否存在指定角色
func (c *CasbinHandler) HasRoleForUser(roleId int, user string) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enforcer.HasRoleForUser(user, c.MakeRoleName(roleId))
}

func (c *CasbinHandler) DeleteRoleForUser(roleId int, user string) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enforcer.DeleteRoleForUser(user, c.MakeRoleName(roleId))
}

// AddUserRole 添加用户角色
func (c *CasbinHandler) AddUserRole(user string, roleId int) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enforcer.AddGroupingPolicy(user, c.MakeRoleName(roleId))
}

// AddUserRoles 批量添加角色和用户的对应关系
func (c *CasbinHandler) AddUserRoles(usernames []string, roleIds []int) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	rules := make([][]string, 0)
	for _, username := range usernames {
		for _, roleId := range roleIds {
			rules = append(rules, []string{username, c.MakeRoleName(roleId)})
		}
	}
	return c.enforcer.AddGroupingPolicies(rules)
}

// DeleteUserRole 删除用户所有角色
func (c *CasbinHandler) DeleteUserRole(user string) (bool, error) {
	c.init()
	if initError != nil {
		return false, initError
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// 删除该用户的所有角色关联（g 策略中 user 是第一个字段）
	return c.enforcer.RemoveFilteredNamedGroupingPolicy("g", 0, user)
}
