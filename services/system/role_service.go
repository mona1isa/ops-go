package system

import (
	"errors"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services"
	"gorm.io/gorm"
	"log"
)

type RoleService struct{}

const DISABLED = "0"

// Add 添加角色
func (r *RoleService) Add(request *api.RoleRequest) error {
	// 校验角色名称是否存在
	name := request.Name
	var count int64
	models.DB.Model(&models.SysRole{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return errors.New("角色名称已存在")
	}
	// 添加角色
	role := &models.SysRole{
		Name:     request.Name,
		OrderNum: request.OrderNum,
		Status:   request.Status,
	}
	role.Remark = request.Remark
	tx := models.DB.Model(&models.SysRole{}).Create(&role)
	if err := tx.Error; err != nil {
		log.Println("添加角色失败：", err.Error())
		return errors.New("添加角色失败：" + err.Error())
	}
	var roleMenus []models.SysRoleMenu
	roleId := role.ID
	// 添加角色权限
	for _, menuId := range request.MenuIds {
		roleMenus = append(roleMenus, models.SysRoleMenu{RoleId: roleId, MenuId: menuId})
	}

	if err := models.DB.Model(models.SysRoleMenu{}).Create(&roleMenus).Error; err != nil {
		log.Println("添加角色权限失败：", err.Error())
		return errors.New("添加角色权限失败：" + err.Error())
	}
	// 将角色关联的菜单权限同步casbin 策略中
	var menus []models.SysMenu
	if err := models.DB.Where("id in ?", request.MenuIds).Find(&menus).Error; err != nil {
		log.Println("查询菜单异常：", err.Error())
		return errors.New("查询菜单异常：" + err.Error())
	}
	if err := saveCasbinPolicy(roleId, menus); err != nil {
		log.Println("保存casbin策略异常：", err.Error())
		return errors.New("保存casbin策略异常：" + err.Error())
	}
	return nil
}

// Edit 编辑角色
func (r *RoleService) Edit(request *api.EditRoleRequest) error {
	id := request.Id
	var count int64
	models.DB.Model(&models.SysRole{}).Where("id = ?", id).Count(&count)
	if count == 0 {
		return errors.New("角色不存在")
	}
	// 编辑角色
	role := &models.SysRole{
		Name:     request.Name,
		OrderNum: request.OrderNum,
		Status:   request.Status,
	}
	role.Remark = request.Remark
	if err := models.DB.Where("id = ?", id).Updates(&role).Error; err != nil {
		log.Println("编辑角色失败：", err.Error())
		return errors.New("编辑角色失败：" + err.Error())
	}
	// 保存角色菜单
	_ = saveRoleMenu(request.MenuIds, id)
	return nil
}

// List 角色列表
func (r *RoleService) List() ([]models.SysRole, error) {
	all, err := services.FindAll[models.SysRole]()
	if err != nil {
		log.Println("查询角色异常：", err)
		return nil, err
	}
	return all, nil
}

// Page 分页查询角色
func (r *RoleService) Page(roleRequest *api.PageRoleRequest) (models.PageResult[models.SysRole], error) {
	pageNum := roleRequest.PageNum
	pageSize := roleRequest.PageSize

	var scopes []func(db *gorm.DB) *gorm.DB
	if roleRequest.Name != "" {
		scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
			return db.Where("name like ?", "%"+roleRequest.Name+"%")
		})
	}
	if roleRequest.Status != "" {
		scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", roleRequest.Status)
		})
	}

	// 根据 orderNum 排序
	scopes = append(scopes, func(db *gorm.DB) *gorm.DB { return db.Order("order_num asc") })

	pageResult, err := models.Paginate[models.SysRole](models.DB, pageNum, pageSize, scopes...)
	if err != nil {
		panic(err)
	}
	return pageResult, nil
}

// Remove 删除角色
func (r *RoleService) Remove(id int) error {
	if err := models.DB.Delete(&models.SysRole{}, id).Error; err != nil {
		return errors.New("角色删除失败: " + err.Error())
	}
	// 删除角色菜单
	if err := models.DB.Where("role_id = ?", id).Delete(&models.SysRoleMenu{}).Error; err != nil {
		return errors.New("删除角色菜单失败: " + err.Error())
	}
	// 删除 Casbin 权限
	_, err := models.Casbin.DeleteRolePolicy(id)
	if err != nil {
		log.Println("删除Casbin 策略失败：", err.Error())
	}
	return nil
}

// GetMenuIds 获取角色菜单
func (r *RoleService) GetMenuIds(roleId int) []int {
	var menuIds []int
	models.DB.Model(models.SysRoleMenu{}).Select("menu_id").Where("role_id = ?", roleId).Find(&menuIds)
	return menuIds
}

// GetUserIds 获取角色用户
func (r *RoleService) GetUserIds(roleId int) []int {
	var userIds []int
	models.DB.Model(models.SysUserRole{}).Select("user_id").Where("role_id = ?", roleId).Find(&userIds)
	return userIds
}

// GetAsignUserInfo 获取角色分配用户信息
func (r *RoleService) GetAsignUserInfo(roleId int) map[string]any {
	var result = make(map[string]any)
	// 已分配用户
	var userIds []int
	models.DB.Model(models.SysUserRole{}).Select("user_id").Where("role_id = ?", roleId).Find(&userIds)

	// 所有用户
	var allUserList []models.SysUser
	models.DB.Where("del_flag = ? and status = ?", "0", "1").Find(&allUserList)

	// 分组
	assignedUsers := make([]map[string]any, 0)
	unassignedUsers := make([]map[string]any, 0)
	for _, user := range allUserList {
		found := false
		for _, id := range userIds {
			if user.ID == id {
				found = true
				break
			}
		}
		userInfo := map[string]any{
			"id":   user.ID,
			"name": user.NickName,
		}
		if found {
			assignedUsers = append(assignedUsers, userInfo)
		} else {
			unassignedUsers = append(unassignedUsers, userInfo)
		}
	}

	result["assigned"] = assignedUsers
	result["unassigned"] = unassignedUsers
	return result
}

// RoleAsignUsers 角色授权
func (r *RoleService) RoleAsignUsers(request api.RoleAsignRequest) error {
	roleId := request.RoleId
	var role models.SysRole
	if err := models.DB.Model(models.SysRole{}).Where("id = ?", roleId).Find(&role).Error; err != nil {
		return errors.New("角色不存在")
	}

	if role.Status == DISABLED {
		return errors.New("角色已禁用，无法授权")
	}

	// 删除旧的角色用户
	_ = saveUserRole(request.UserIds, roleId)
	return nil
}

// saveRoleMenu 保存角色菜单
func saveRoleMenu(menuIds []int, roleId int) error {
	if len(menuIds) == 0 {
		return nil
	}
	models.DB.Model(&models.SysRoleMenu{}).Where("role_id = ?", roleId).Delete(&models.SysRoleMenu{})
	for _, menuId := range menuIds {
		if err := models.DB.Model(&models.SysRoleMenu{}).Create(&models.SysRoleMenu{RoleId: roleId, MenuId: menuId}).Error; err != nil {
			return errors.New("保存角色菜单失败: " + err.Error())
		}
	}

	var menus []models.SysMenu
	if err := models.DB.Where("id in ?", menuIds).Find(&menus).Error; err != nil {
		return errors.New("查询菜单失败: " + err.Error())
	}
	// 将角色关联的菜单权限同步casbin 策略中
	err := saveCasbinPolicy(roleId, menus)
	if err != nil {
		log.Println("保存casbin策略失败: " + err.Error())
		return nil
	}
	return nil
}

// saveUserRole 保存用户角色
func saveUserRole(userIds []int, roleId int) error {
	// 删除旧的角色用户
	models.DB.Model(&models.SysUserRole{}).Where("role_id = ?", roleId).Delete(&models.SysUserRole{})
	// 同步删除 Casbin 权限
	_, _ = models.Casbin.DeleteRole(roleId)

	// 添加新的角色用户
	if len(userIds) > 0 {
		var userRoles []models.SysUserRole
		for _, userId := range userIds {
			userRoles = append(userRoles, models.SysUserRole{RoleId: roleId, UserId: userId})
		}
		models.DB.Create(&userRoles)

		// 同步 Casbin 用户角色
		var users []models.SysUser
		_ = models.DB.Where("id in ?", userIds).Find(&users).Error
		var names []string
		for _, user := range users {
			names = append(names, user.UserName)
		}
		roles, err := models.Casbin.AddUserRoles(names, []int{roleId})
		if err != nil {
			log.Println(roles)
			log.Println("添加用户角色同步Casbin异常：", err)
		}
	}

	return nil
}

func saveCasbinPolicy(roleId int, menus []models.SysMenu) error {
	// 删除旧的角色策略
	_, err := models.Casbin.DeleteRolePolicy(roleId)
	if err != nil {
		log.Fatalln("删除旧的角色策略失败：", err.Error())
	}
	// 添加新的角色策略
	for _, menu := range menus {
		url := menu.RequestUrl
		method := menu.RequestMethod
		if url != "" && method != "" {
			_, err := models.Casbin.AddPolicy(roleId, url, method)
			if err != nil {
				log.Fatalln("添加策略失败：", err.Error())
				return err
			}
		}
	}

	return nil
}
