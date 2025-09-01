package system

import (
	"errors"
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
	"log"
)

type MenuService struct{}

// Add 添加菜单
func (*MenuService) Add(request *api.AddMenuRequest) error {
	name := request.Name
	if name != "" {
		var count int64
		config.DB.Model(models.SysMenu{}).Where(NameScope(name)).Count(&count)
		if count > 0 {
			return errors.New("菜单名称已存在")
		}
	}

	parentId := request.ParentId
	if parentId != 0 {
		var count int64
		config.DB.Model(models.SysMenu{}).Where("id=?", parentId).Count(&count)
		if count == 0 {
			return errors.New("父菜单不存在")
		}
	}

	menu := models.SysMenu{
		Name:      request.Name,
		ParentId:  request.ParentId,
		OrderNum:  request.OrderNum,
		Path:      request.Path,
		Component: request.Component,
		IsFrame:   request.IsFrame,
		IsCache:   request.IsCache,
		Type:      request.Type,
		Visible:   request.Visible,
		Status:    request.Status,
		Perms:     request.Perms,
		Icon:      request.Icon,
	}
	menu.CreateBy = request.CreateBy
	menu.UpdateBy = request.UpdateBy
	if err := config.DB.Model(models.SysMenu{}).Create(&menu).Error; err != nil {
		log.Println("新增菜单失败", err)
		return errors.New("新增菜单失败")
	}

	return nil
}

// List 菜单列表
func (*MenuService) List(request *api.MenuListRequest) ([]*api.MenuTree, error) {
	menuList := make([]models.SysMenu, 0)

	query := config.DB.Model(models.SysMenu{})
	var scopes []func(db *gorm.DB) *gorm.DB
	name := request.Name
	if name != "" {
		scopes = append(scopes, NameScope(name))
	}

	status := request.Status
	if status != "" {
		scopes = append(scopes, StatusScope(status))
	}

	scopes = append(scopes, DelFlagScope("0"))
	if len(scopes) > 0 {
		query = query.Scopes(scopes...)
	}

	if err := query.Find(&menuList).Error; err != nil {
		log.Println("查询菜单列表失败", err)
		return nil, errors.New("查询菜单列表失败")
	}

	result := BuildMenuTree(menuList, 0)
	return result, nil
}

// Edit 编辑菜单
func (*MenuService) Edit() error {
	return nil
}

// Delete 删除菜单
func (*MenuService) Delete() error {
	return nil
}

// NameScope 构建名称查询条件
func NameScope(name string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("name like ?", "%"+name+"%")
	}
}

// StatusScope 构建状态查询条件
func StatusScope(status string) func(db *gorm.DB) *gorm.DB {
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
	return &api.MenuTree{
		Id:        menu.ID,
		Name:      menu.Name,
		ParentId:  menu.ParentId,
		OrderNum:  menu.OrderNum,
		Path:      menu.Path,
		Component: menu.Component,
		IsFrame:   menu.IsFrame,
		IsCache:   menu.IsCache,
		Type:      menu.Type,
		Visible:   menu.Visible,
		Status:    menu.Status,
		Perms:     menu.Perms,
		Icon:      menu.Icon,
	}
}
