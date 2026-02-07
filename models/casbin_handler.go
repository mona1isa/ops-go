package models

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"sync"
	"time"
)

var (
	once   sync.Once
	Casbin *CasbinHandler
)

type CasbinHandler struct {
	enforcer *casbin.SyncedEnforcer
}

func (c *CasbinHandler) init() {
	once.Do(func() {
		adapter, err := gormadapter.NewAdapterByDB(DB)
		if err != nil {
			panic(err)
		}
		c.enforcer, err = casbin.NewSyncedEnforcer("config/casbin_rbac.conf", adapter)
		if err != nil {
			panic(err)
		}
		// 配置自动刷新策略间隔（30秒）
		c.enforcer.StartAutoLoadPolicy(30 * time.Second)
		// 启用自动保存策略
		c.enforcer.EnableAutoSave(true)
		// 仅在开发环境启用日志
		// c.enforcer.EnableLog(true)
		// 加载策略
		err = c.enforcer.LoadPolicy()
		if err != nil {
			panic(fmt.Sprintf("Failed to load policy: %v", err))
		}
	})
}

// Enforcer Casbin权限验证
func (c *CasbinHandler) Enforcer(user, uri, method string) (bool, error) {
	return c.enforcer.Enforce(user, uri, method)
}

// AddPolicy 添加策略
func (c *CasbinHandler) AddPolicy(roleId int, uri, method string) (bool, error) {
	return c.enforcer.AddPolicy(c.MakeRoleName(roleId), uri, method)
}

// MakeRoleName 拼接角色ID，为了防止角色与用户名冲突
func (c *CasbinHandler) MakeRoleName(roleId int) string {
	return fmt.Sprintf("role_%d", roleId)
}

// AddPolicies 添加策略
func (c *CasbinHandler) AddPolicies(rules [][]string) (bool, error) {
	return c.enforcer.AddPolicies(rules)
}

// DeleteRolePolicy 删除角色下的权限
func (c *CasbinHandler) DeleteRolePolicy(roleId int) (bool, error) {
	return c.enforcer.RemoveFilteredNamedPolicy("p", 0, c.MakeRoleName(roleId))
}

// DeleteRoleById 根据角色ID删除角色（包含p和g）
func (c *CasbinHandler) DeleteRoleById(roleId int) (bool, error) {
	return c.enforcer.DeleteRole(c.MakeRoleName(roleId))
}

// DeleteRole 删除角色
func (c *CasbinHandler) DeleteRole(roleId int) (bool, error) {
	return c.enforcer.RemoveFilteredNamedPolicy("g", 1, c.MakeRoleName(roleId))
}

// ClearUserRole 清除用户角色
func (c *CasbinHandler) ClearUserRole(roleId int, user string) (bool, error) {
	return c.enforcer.RemoveFilteredNamedGroupingPolicy("g", 1, c.MakeRoleName(roleId), user)
}

// HasRoleForUser 判断用户是否存在指定角色
func (c *CasbinHandler) HasRoleForUser(roleId int, user string) (bool, error) {
	return c.enforcer.HasRoleForUser(user, c.MakeRoleName(roleId))
}

func (c *CasbinHandler) DeleteRoleForUser(roleId int, user string) (bool, error) {
	return c.enforcer.DeleteRoleForUser(c.MakeRoleName(roleId), user)
}

// AddUserRole 添加用户角色
func (c *CasbinHandler) AddUserRole(user string, roleId int) (bool, error) {
	return c.enforcer.AddGroupingPolicy(user, c.MakeRoleName(roleId))
}

// AddUserRoles 批量添加角色和用户的对应关系
func (c *CasbinHandler) AddUserRoles(usernames []string, roleIds []int) (bool, error) {
	rules := make([][]string, 0)
	for _, username := range usernames {
		for _, roleId := range roleIds {
			rules = append(rules, []string{username, c.MakeRoleName(roleId)})
		}
	}
	return c.enforcer.AddGroupingPolicies(rules)
}

// DeleteUserRole 删除用户角色
func (c *CasbinHandler) DeleteUserRole(user string) (bool, error) {
	return c.enforcer.RemoveFilteredNamedGroupingPolicy("g", 0, user)
}
