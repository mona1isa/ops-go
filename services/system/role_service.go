package system

import (
	"errors"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services"
	"gorm.io/gorm"
	"log"
	"strings"
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
	// 保存角色菜单（包括 Casbin 同步）
	if err := saveRoleMenu(request.MenuIds, id); err != nil {
		log.Println("保存角色菜单失败：", err.Error())
		return errors.New("保存角色菜单失败：" + err.Error())
	}
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
	if err := models.DB.Transaction(func(tx *gorm.DB) error {
		// 删除角色
		if err := tx.Delete(&models.SysRole{}, id).Error; err != nil {
			return errors.New("角色删除失败: " + err.Error())
		}
		// 删除角色菜单
		if err := tx.Where("role_id = ?", id).Delete(&models.SysRoleMenu{}).Error; err != nil {
			return errors.New("删除角色菜单失败: " + err.Error())
		}
		// 删除角色用户关联
		if err := tx.Where("role_id = ?", id).Delete(&models.SysUserRole{}).Error; err != nil {
			return errors.New("删除角色用户关联失败: " + err.Error())
		}
		// 删除 Casbin 权限
		_, err := models.Casbin.DeleteRoleById(id)
		if err != nil {
			log.Printf("删除Casbin 策略失败：%v\n", err)
			return errors.New("删除Casbin 策略失败: " + err.Error())
		}
		return nil
	}); err != nil {
		return err
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
	var count int64
	models.DB.Model(&models.SysRole{}).Where("id = ?", roleId).Count(&count)
	if count == 0 {
		return result
	}

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
	_ = saveUserRole(request.AddedUserIds, request.RemovedUserIds, roleId)
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
func saveUserRole(addedUserIds []int, removedUserIds []int, roleId int) error {
	// 删除用户角色
	if len(removedUserIds) > 0 {
		models.DB.Where("role_id = ? and user_id in ?", roleId, removedUserIds).Delete(&models.SysUserRole{})

		var users []models.SysUser
		models.DB.Where("id in ?", removedUserIds).Find(&users)
		for _, user := range users {
			_, err := models.Casbin.DeleteRoleForUser(roleId, user.UserName)
			if err != nil {
				log.Printf("删除用户角色同步Casbin异常：username=%s, err=%s\n", user.UserName, err)
			}
		}
	}

	// 添加新的角色用户
	if len(addedUserIds) > 0 {
		var userRoles []models.SysUserRole
		for _, userId := range addedUserIds {
			userRoles = append(userRoles, models.SysUserRole{RoleId: roleId, UserId: userId})
		}
		models.DB.Create(&userRoles)

		// 同步 Casbin 用户角色
		var users []models.SysUser
		_ = models.DB.Where("id in ?", addedUserIds).Find(&users).Error
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

// saveCasbinPolicy 保存角色的 Casbin 策略
// 先添加新策略，成功后删除旧策略，确保原子性
func saveCasbinPolicy(roleId int, menus []models.SysMenu) error {
	// 构建新的策略列表
	newPolicies := policiesOfMenus(models.Casbin.MakeRoleName(roleId), menus)

	// 获取旧策略（用于失败时回滚）
	oldPolicies, err := getRolePolicies(roleId)
	if err != nil {
		log.Printf("获取旧策略失败：%v\n", err)
		// 继续执行，尝试清空后重新添加
	}

	// 先删除旧策略
	_, err = models.Casbin.DeleteRolePolicy(roleId)
	if err != nil {
		log.Printf("删除旧的角色策略失败：%v\n", err)
		return err
	}

	// 添加新策略
	if len(newPolicies) > 0 {
		_, err = models.Casbin.AddPolicies(newPolicies)
		if err != nil {
			log.Printf("添加新策略失败：%v\n", err)
			// 尝试回滚：恢复旧策略
			if len(oldPolicies) > 0 {
				_, rollbackErr := models.Casbin.AddPolicies(oldPolicies)
				if rollbackErr != nil {
					log.Printf("回滚策略失败：%v\n", rollbackErr)
				}
			}
			return err
		}
	}

	return nil
}

// getRolePolicies 获取角色的现有策略（用于回滚）
func getRolePolicies(roleId int) ([][]string, error) {
	menus, err := roleAssignedMenus(roleId)
	if err != nil {
		return nil, err
	}
	return policiesOfMenus(models.Casbin.MakeRoleName(roleId), menus), nil
}

// roleAssignedMenus 角色已勾选的菜单
func roleAssignedMenus(roleId int) ([]models.SysMenu, error) {
	var menus []models.SysMenu
	var roleMenus []models.SysRoleMenu
	if err := models.DB.Where("role_id = ?", roleId).Find(&roleMenus).Error; err != nil {
		return nil, err
	}

	menuIds := make([]int, 0)
	for _, rm := range roleMenus {
		menuIds = append(menuIds, rm.MenuId)
	}

	if len(menuIds) == 0 {
		return menus, nil
	}

	if err := models.DB.Where("id in ?", menuIds).Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
}

// policiesOfMenus 菜单授权对应的 Casbin 策略行，只有接口地址与方法齐全的菜单才产生策略。
// 同一接口可能被多个菜单引用（页面与其按钮同址），这里去重：重复行会让 gorm-adapter 的
// 批量写入触发 MySQL 1869（Auto-increment 冲突），整批策略写入失败。
func policiesOfMenus(roleName string, menus []models.SysMenu) [][]string {
	policies := make([][]string, 0, len(menus))
	seen := make(map[string]bool, len(menus))
	for _, menu := range menus {
		if menu.RequestUrl == "" || menu.RequestMethod == "" {
			continue
		}
		key := menu.RequestUrl + "\t" + menu.RequestMethod
		if seen[key] {
			continue
		}
		seen[key] = true
		policies = append(policies, []string{roleName, menu.RequestUrl, menu.RequestMethod})
	}
	return policies
}

// SyncRolePolicies 按 sys_role_menu 重建各角色的 Casbin 策略，仅重写有差异的角色。
// 背景：策略只在管理员保存角色权限时写入，新功能菜单由启动种子补写后，
// 非超管角色即便已被授予菜单也没有对应的接口策略（请求 403），前端按钮也缺少权限标识。
func SyncRolePolicies() {
	var roleIds []int
	if err := models.DB.Table(models.TableSysRoleMenu).
		Select("DISTINCT role_id").Scan(&roleIds).Error; err != nil {
		log.Println("同步角色 Casbin 策略失败：查询已授权角色异常：", err)
		return
	}

	for _, roleId := range roleIds {
		menus, err := roleAssignedMenus(roleId)
		if err != nil {
			log.Printf("同步角色 %d 的 Casbin 策略失败：查询菜单异常：%v\n", roleId, err)
			continue
		}
		expected := policiesOfMenus(models.Casbin.MakeRoleName(roleId), menus)
		current, err := models.Casbin.GetRolePolicies(roleId)
		if err != nil {
			log.Printf("同步角色 %d 的 Casbin 策略失败：读取现有策略异常：%v\n", roleId, err)
			continue
		}
		if samePolicies(expected, current) {
			continue
		}
		if err := saveCasbinPolicy(roleId, menus); err != nil {
			log.Printf("同步角色 %d 的 Casbin 策略失败：%v\n", roleId, err)
			continue
		}
		log.Printf("角色 %d 的 Casbin 策略已同步：%d 条（原 %d 条）\n", roleId, len(expected), len(current))
	}
}

func samePolicies(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]struct{}, len(a))
	for _, rule := range a {
		seen[strings.Join(rule, "\t")] = struct{}{}
	}
	for _, rule := range b {
		if _, ok := seen[strings.Join(rule, "\t")]; !ok {
			return false
		}
	}
	return true
}

