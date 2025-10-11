package system

import (
	"errors"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services"
	"gorm.io/gorm"
	"log"
)

type MenuService struct {
}

// Add 添加菜单
func (*MenuService) Add(request *api.AddMenuRequest) error {
	name := request.Name
	if name != "" {
		var count int64
		models.DB.Model(models.SysMenu{}).Where("name = ?", name).Count(&count)
		if count > 0 {
			return errors.New("菜单名称已存在")
		}
	}

	parentId := request.ParentId
	if parentId != 0 {
		var count int64
		models.DB.Model(models.SysMenu{}).Where("id=?", parentId).Count(&count)
		if count == 0 {
			return errors.New("父菜单不存在")
		}
	}

	menu := models.SysMenu{
		Name:          request.Name,
		ParentId:      request.ParentId,
		OrderNum:      request.OrderNum,
		Path:          request.Path,
		Component:     request.Component,
		IsAffix:       request.IsAffix,
		IsIframe:      request.IsIframe,
		IsLink:        request.IsLink,
		KeepAlive:     request.KeepAlive,
		Type:          request.Type,
		IsHide:        request.IsHide,
		Status:        request.Status,
		Perms:         request.Perms,
		Icon:          request.Icon,
		Url:           request.Url,
		RequestUrl:    request.RequestUrl,
		RequestMethod: request.RequestMethod,
	}
	menu.CreateBy = request.CreateBy
	menu.UpdateBy = request.UpdateBy
	if err := models.DB.Model(models.SysMenu{}).Create(&menu).Error; err != nil {
		log.Println("新增菜单失败", err)
		return errors.New("新增菜单失败")
	}

	return nil
}

// RoutesList 前段获取路由信息
func (*MenuService) RoutesList(userId string, isAdmin bool) ([]*api.MenuTree, error) {
	menuList := make([]models.SysMenu, 0)

	if isAdmin {
		tx := models.DB.Model(models.SysMenu{}).
			Where("type <> ? AND status = ? AND del_flag = ? order by order_num asc", "F", "1", "0")
		if err := tx.Find(&menuList).Error; err != nil {
			log.Println("查询菜单列表失败", err)
			return nil, errors.New("查询菜单列表失败")
		}
	} else {
		var userRoles []models.SysUserRole
		if err := models.DB.Model(models.SysUserRole{}).Where("user_id = ?", userId).Find(&userRoles).Error; err != nil {
			log.Println("查询用户角色失败", err)
			return nil, errors.New("查询用户角色失败")
		}

		var roleIds []int
		for _, roleId := range userRoles {
			roleIds = append(roleIds, roleId.RoleId)
		}
		// 查询角色关联的菜单
		var roleMenu []models.SysRoleMenu
		if err := models.DB.Model(models.SysRoleMenu{}).Where("role_id in ?", roleIds).Find(&roleMenu).Error; err != nil {
			log.Println("查询角色关联的菜单失败", err)
			return nil, errors.New("查询角色关联的菜单失败")
		}
		var menuIds []int
		for _, v := range roleMenu {
			menuIds = append(menuIds, v.MenuId)
		}

		// 查询菜单
		tx := models.DB.Model(models.SysMenu{}).
			Where("type <> ? AND status = ? AND del_flag = ? AND id IN ? order by order_num asc", "F", "1", "0", menuIds)
		if err := tx.Find(&menuList).Error; err != nil {
			log.Println("查询菜单列表失败", err)
			return nil, errors.New("查询菜单列表失败")
		}
	}

	result := BuildMenuTree(menuList, 0)
	return result, nil
}

// List 菜单列表
func (*MenuService) List(request *api.MenuListRequest) ([]*api.MenuTree, error) {
	menuList := make([]models.SysMenu, 0)

	query := models.DB.Model(models.SysMenu{})
	var scopes []func(db *gorm.DB) *gorm.DB
	name := request.Name
	if name != "" {
		scopes = append(scopes, NameLikeScope(name))
	}

	scopes = append(scopes, StatusScope(true))

	scopes = append(scopes, DelFlagScope("0"))
	if len(scopes) > 0 {
		query = query.Scopes(scopes...)
	}

	query = query.Order("order_num asc")
	if err := query.Find(&menuList).Error; err != nil {
		log.Println("查询菜单列表失败", err)
		return nil, errors.New("查询菜单列表失败")
	}

	result := BuildMenuTree(menuList, 0)
	return result, nil
}

// Edit 编辑菜单
func (*MenuService) Edit(request *api.EditMenuRequest) error {
	id := request.Id
	_, err := services.FindById[models.SysMenu](id)
	if err != nil {
		log.Println("查询菜单失败", err)
		return errors.New("查询菜单失败")
	}

	var count int64
	models.DB.Model(models.SysMenu{}).Where("name = ? and id <> ?", request.Name, id).Count(&count)
	if count > 0 {
		return errors.New("菜单名称已存在")
	}

	updates := map[string]any{
		"parent_id":      request.ParentId,
		"name":           request.Name,
		"order_num":      request.OrderNum,
		"path":           request.Path,
		"component":      request.Component,
		"is_affix":       request.IsAffix,
		"is_iframe":      request.IsIframe,
		"is_link":        request.IsLink,
		"keep_alive":     request.KeepAlive,
		"type":           request.Type,
		"is_hide":        request.IsHide,
		"status":         request.Status,
		"perms":          request.Perms,
		"icon":           request.Icon,
		"url":            request.Url,
		"request_url":    request.RequestUrl,
		"request_method": request.RequestMethod,
	}
	err = services.Update[models.SysMenu](id, updates)
	return nil
}

// Delete 删除菜单
func (m *MenuService) Delete(id int) error {
	err := services.Delete[models.SysMenu](id)
	if err != nil {
		log.Println("删除菜单错误：", err)
		return errors.New("删除菜单错误")
	}
	return nil
}

// NameEqScope 构建名称精确查询条件
func NameEqScope(name string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", name)
	}
}

// NameScope 构建名称模糊查询条件
func NameLikeScope(name string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("name like ?", "%"+name+"%")
	}
}

// StatusScope 构建状态查询条件
func StatusScope(status bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", status)
	}
}

func DelFlagScope(delFlag string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("del_flag = ?", delFlag)
	}
}

// BuildMenuTree 构建菜单树
func BuildMenuTree(menuList []models.SysMenu, parent int) []*api.MenuTree {
	var tree []*api.MenuTree
	for _, menu := range menuList {
		if menu.ParentId == parent {
			node := ConvertToDto(menu)
			children := BuildMenuTree(menuList, menu.ID)
			if len(children) > 0 {
				node.Children = children
			}
			tree = append(tree, node)
		}
	}
	return tree
}

// ConvertToDto 转换为DTO
func ConvertToDto(menu models.SysMenu) *api.MenuTree {
	meta := api.Meta{
		KeepAlive: menu.KeepAlive,
		Title:     menu.Name,
		IsLink:    menu.IsLink,
		IsHide:    menu.IsHide,
		IsAffix:   menu.IsAffix,
		IsIframe:  menu.IsIframe,
		Roles:     []string{"admin"},
		Icon:      menu.Icon,
	}
	if menu.Type == "C" {
		meta.KeepAlive = true
	} else {
		meta.KeepAlive = false
	}

	return &api.MenuTree{
		Id:            menu.ID,
		Name:          menu.Name,
		ParentId:      menu.ParentId,
		OrderNum:      menu.OrderNum,
		Path:          menu.Path,
		Component:     menu.Component,
		Type:          menu.Type,
		Status:        menu.Status,
		Perms:         menu.Perms,
		Icon:          menu.Icon,
		Url:           menu.Url,
		RequestUrl:    menu.RequestUrl,
		RequestMethod: menu.RequestMethod,
		Meta:          meta,
	}
}
