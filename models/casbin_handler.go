package models

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"sync"
)

var (
	once   sync.Once
	Casbin *CasbinHandler
)

type CasbinHandler struct {
	syncedEnforcer *casbin.SyncedEnforcer
}

func (c *CasbinHandler) init() {
	once.Do(func() {
		adapter, err := gormadapter.NewAdapterByDB(DB)
		if err != nil {
			panic(err)
		}
		c.syncedEnforcer, err = casbin.NewSyncedEnforcer("config/casbin_rbac.conf", adapter)
		if err != nil {
			panic(err)
		}
	})
	c.syncedEnforcer.AddFunction("isAdmin", func(arguments ...interface{}) (interface{}, error) {
		username := arguments[0].(string)
		return c.syncedEnforcer.HasRoleForUser(username, "admin")
	})

	err := c.syncedEnforcer.LoadPolicy()
	if err != nil {
		panic(err)
	}
}

// Enforcer Casbin权限验证
func (c *CasbinHandler) Enforcer(user, uri, method string) (bool, error) {
	return c.syncedEnforcer.Enforce(user, uri, method)
}

// AddPolicy 添加策略
func (c *CasbinHandler) AddPolicy(roleId int, uri, method string) (bool, error) {
	return c.syncedEnforcer.AddPolicy(c.MakeRoleName(roleId), uri, method)
}

// MakeRoleName 拼接角色ID，为了防止角色与用户名冲突
func (c *CasbinHandler) MakeRoleName(roleId int) string {
	return fmt.Sprintf("role_%d", roleId)
}

// AddPolicies 添加策略
func (c *CasbinHandler) AddPolicies(rules [][]string) (bool, error) {
	return c.syncedEnforcer.AddPolicies(rules)
}

func (c *CasbinHandler) DeleteRole(roleId int) (bool, error) {
	return c.syncedEnforcer.DeleteRole(c.MakeRoleName(roleId))
}

// DeleteRolePolicy 删除角色下的权限
func (c *CasbinHandler) DeleteRolePolicy(roleId int) (bool, error) {
	return c.syncedEnforcer.RemoveFilteredNamedPolicy("p", 0, c.MakeRoleName(roleId))
}

// DeleteRoleUser 删除角色下的用户
func (c *CasbinHandler) DeleteRoleUser(roleId int) (bool, error) {
	return c.syncedEnforcer.RemoveFilteredNamedPolicy("g", 1, c.MakeRoleName(roleId))
}

// ClearUserRole 清除用户角色
func (c *CasbinHandler) ClearUserRole(roleId int, user string) (bool, error) {
	return c.syncedEnforcer.RemoveFilteredNamedGroupingPolicy("g", 1, c.MakeRoleName(roleId), user)
}

// HasRoleForUser 判断用户是否存在指定角色
func (c *CasbinHandler) HasRoleForUser(roleId int, user string) (bool, error) {
	return c.syncedEnforcer.HasRoleForUser(user, c.MakeRoleName(roleId))
}

func (c *CasbinHandler) DeleteRoleForUser(roleId int, user string) (bool, error) {
	return c.syncedEnforcer.DeleteRoleForUser(c.MakeRoleName(roleId), user)
}

// AddUserRole 添加用户角色
func (c *CasbinHandler) AddUserRole(user string, roleId int) (bool, error) {
	return c.syncedEnforcer.AddGroupingPolicy(user, c.MakeRoleName(roleId))
}

// AddUserRoles 批量添加角色和用户的对应关系
func (c *CasbinHandler) AddUserRoles(usernames []string, roleIds []int) (bool, error) {
	rules := make([][]string, 0)
	for _, username := range usernames {
		for _, roleId := range roleIds {
			rules = append(rules, []string{username, c.MakeRoleName(roleId)})
		}
	}
	return c.syncedEnforcer.AddGroupingPolicies(rules)
}

// DeleteUserRole 删除用户角色
func (c *CasbinHandler) DeleteUserRole(user string) (bool, error) {
	return c.syncedEnforcer.RemoveFilteredNamedGroupingPolicy("g", 0, user)
}
