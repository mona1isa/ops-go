package instance

import (
	"errors"
	"fmt"
	"github.com/zhany/ops-go/models"
	"log"
	"math"
)

type UserInstanceAuth struct {
	UserId      int   `json:"userId"`
	InstanceIds []int `json:"instanceIds"`
	GroupIds    []int `json:"groupIds"`
	AuthType    int   `json:"authType"` // 类型: 1 主机 2 分组 3 同时存在主机和分组
}

type PageUserInstanceAuth struct {
	UserId   int `json:"userId"`
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

type UserInstanceAuthService interface {
	CreateUserInstanceAuth(userInstanceAuth *UserInstanceAuth) error
	DeleteUserInstanceAuth(userInstanceAuth *UserInstanceAuth) error
	GetUserInstanceAuth(userId int) (map[string]any, error)
	GetUserInstances(userId int) ([]models.OpsInstance, error)
	GetUserInstancesPage(pageUserInstanceAuth *PageUserInstanceAuth) (map[string]any, error)
	GetInstances(pageUserInstanceAuth *PageUserInstanceAuth) (map[string]any, error)
	GetGroups(pageUserInstanceAuth *PageUserInstanceAuth) (map[string]any, error)
}

// CreateUserInstanceAuth 创建用户-主机/分组授权关系
func (auth *UserInstanceAuth) CreateUserInstanceAuth() error {
	userId := auth.UserId
	instanceIds := auth.InstanceIds
	groupIds := auth.GroupIds
	authType := auth.AuthType

	if userId == 0 {
		return errors.New("用户ID不能为空")
	}

	if authType == 0 {
		return errors.New("类型不能为空")
	}
	if len(instanceIds) == 0 && len(groupIds) == 0 {
		log.Println("主机和分组ID不能同时为空")
		return errors.New("请选择主机，主机分组进行授权")
	}

	// 校验用户是否存在
	if err := models.DB.First(&models.SysUser{}, userId).Error; err != nil {
		log.Println("用户不存在:", err)
		return errors.New("用户不存在")
	}
	// 校验主机/分组是否存在
	if authType == 1 && len(instanceIds) > 0 {
		err := createUserInstance(userId, instanceIds, authType)
		if err != nil {
			return err
		}
	} else if authType == 2 && len(groupIds) > 0 {
		err := createUserGroupInstance(userId, groupIds, authType)
		if err != nil {
			return err
		}
	} else if authType == 3 {
		if len(instanceIds) > 0 {
			err := createUserInstance(userId, instanceIds, 1)
			if err != nil {
				return err
			}
		}
		if len(groupIds) > 0 {
			err := createUserGroupInstance(userId, groupIds, 2)
			if err != nil {
				return err
			}
		}
	} else {
		log.Println("授权类型错误")
		return errors.New("授权类型错误")
	}

	return nil
}

// CreateUserInstance 处理用户-主机/分组授权关系
func createUserInstance(userId int, instanceIds []int, authType int) error {
	var instances []models.OpsInstance
	if err := models.DB.Where("id in (?)", instanceIds).Find(&instances).Error; err != nil {
		errInfo := fmt.Sprintf("查询主机信息异常Id: %v Error: %s", instanceIds, err)
		log.Println(errInfo)
		return errors.New("查询主机信息异常")
	}
	if len(instances) != len(instanceIds) {
		// 提取出 instances 里面的主机ID组成一个数组
		ids := make([]int, 0)
		for _, instance := range instances {
			exist := false
			for _, id := range instanceIds {
				if id == instance.ID {
					exist = true
					break
				}
			}
			if !exist {
				ids = append(ids, instance.ID)
			}
		}
		log.Println("主机信息缺失, 主机ID: ", ids)
		return errors.New("主机信息缺失, 请检查后再试")
	}

	// 创建用户-主机授权关系
	var userInstanceAuths []models.OpsUserInstanceAuth
	for _, instanceId := range instanceIds {
		userInstanceAuths = append(userInstanceAuths, models.OpsUserInstanceAuth{
			UserId:     userId,
			InstanceId: instanceId,
			AuthType:   authType,
		})
	}
	if err := models.DB.Create(&userInstanceAuths).Error; err != nil {
		errInfo := fmt.Sprintf("创建用户-主机授权关系异常: %s", err)
		log.Println(errInfo)
		return errors.New("创建用户-主机授权关系异常")
	}
	return nil
}

// CreateUserGroupInstance 处理用户-分组授权关系
func createUserGroupInstance(userId int, groupIds []int, authType int) error {
	var groups []models.OpsGroup
	if err := models.DB.Where("id in (?)", groupIds).Find(&groups).Error; err != nil {
		errInfo := fmt.Sprintf("查询分组信息异常Id: %v Error: %s", groupIds, err)
		log.Println(errInfo)
		return errors.New("查询分组信息异常")
	}
	if len(groups) != len(groupIds) {
		// 提取出 groups 里面的分组ID组成一个数组
		ids := make([]int, 0)
		for _, group := range groups {
			exist := false
			for _, id := range groupIds {
				if id == group.ID {
					exist = true
					break
				}
			}
			if !exist {
				ids = append(ids, group.ID)
			}
		}
		log.Println("主机分组信息缺失, 分组ID: ", ids)
		return errors.New("主机分组信息缺失, 请检查后再试")
	}
	// 创建用户-分组授权关系
	var userInstanceAuths []models.OpsUserInstanceAuth
	for _, groupId := range groupIds {
		userInstanceAuths = append(userInstanceAuths, models.OpsUserInstanceAuth{
			UserId:   userId,
			GroupId:  groupId,
			AuthType: authType,
		})
	}
	if err := models.DB.Create(&userInstanceAuths).Error; err != nil {
		errInfo := fmt.Sprintf("创建用户-主机分组授权关系异常: %s", err)
		log.Println(errInfo)
		return errors.New("创建用户-主机分组授权关系异常")
	}
	return nil
}

// GetUserInstanceAuth 根据 authType 获取用户已授权主机信息，已授权分组信息
func (auth *UserInstanceAuth) GetUserInstanceAuth() (map[string]any, error) {
	var result map[string]any

	userId := auth.UserId
	if userId == 0 {
		return result, errors.New("用户ID不能为空")
	}

	var instances []models.OpsInstance
	if err := models.DB.Table("ops_instance").Select("ops_instance.*").Joins("JOIN ops_user_instance_auth ON ops_instance.id = ops_user_instance_auth.instance_id").Where("ops_user_instance_auth.user_id = ? AND ops_user_instance_auth.auth_type = 1", userId).Find(&instances).Error; err != nil {
		log.Println("获取用户授权主机信息异常: ", err)
		return result, errors.New("获取用户授权主机信息异常")
	}

	var groups []models.OpsGroup
	if err := models.DB.Table("ops_group").Select("ops_group.*").Joins("JOIN ops_user_instance_auth ON ops_group.id = ops_user_instance_auth.group_id").Where("ops_user_instance_auth.user_id = ? AND ops_user_instance_auth.auth_type = 2", userId).Find(&groups).Error; err != nil {
		log.Println("获取用户授权分组信息异常: ", err)
		return result, errors.New("获取用户授权分组信息异常")
	}
	result = map[string]any{
		"instances": instances,
		"groups":    groups,
	}
	return result, nil
}

// GetUserInstances 获取用户授权的主机信息
func (auth *UserInstanceAuth) GetUserInstances() (instances []models.OpsInstance, err error) {
	userId := auth.UserId
	if userId == 0 {
		return instances, errors.New("用户ID不能为空")
	}

	var userInstanceAuths []models.OpsUserInstanceAuth
	if err := models.DB.Where("user_id = ?", userId).Find(&userInstanceAuths).Error; err != nil {
		log.Println("获取用户-主机/分组授权关系异常: ", err)
		return instances, errors.New("获取用户-主机/分组授权关系异常")
	}

	var instanceIds []int
	var groupIds []int
	for _, userInstanceAuth := range userInstanceAuths {
		if userInstanceAuth.AuthType == 1 {
			instanceIds = append(instanceIds, userInstanceAuth.InstanceId)
		} else if userInstanceAuth.AuthType == 2 {
			groupIds = append(groupIds, userInstanceAuth.GroupId)
		}
	}

	var bindInstances []int
	if len(groupIds) > 0 {
		if err := models.DB.Model(&models.OpsInstanceGroup{}).Where("group_id in (?)", groupIds).Select("instance_id").Find(&bindInstances).Error; err != nil {
			log.Println("获取主机分组关联关系异常: ", err)
			return instances, errors.New("获取主机分组关联关系异常")
		}
	}

	// 合并主机ID
	instanceIds = append(instanceIds, bindInstances...)
	if err := models.DB.Model(&models.OpsInstance{}).Where("id in (?)", instanceIds).Find(&instances).Error; err != nil {
		log.Println("获取主机信息异常: ", err)
		return instances, errors.New("获取主机信息异常")
	}
	return instances, nil
}

// DeleteUserInstanceAuth 删除用户-主机/分组授权关系
func (auth *UserInstanceAuth) DeleteUserInstanceAuth() error {
	userId := auth.UserId
	instanceIds := auth.InstanceIds
	groupIds := auth.GroupIds
	authType := auth.AuthType

	if userId == 0 {
		return errors.New("用户ID不能为空")
	}

	if authType == 0 {
		return errors.New("类型不能为空")
	}

	if len(instanceIds) == 0 && len(groupIds) == 0 {
		return errors.New("请选择主机或主机分组进行删除")
	}

	if authType == 1 && len(instanceIds) > 0 {
		if err := models.DB.Delete(&models.OpsUserInstanceAuth{}, "user_id = ? and auth_type = ? and instance_id in (?)", userId, authType, instanceIds).Error; err != nil {
			log.Println("删除用户-主机授权关系异常: ", err)
			return errors.New("删除用户-主机授权关系异常")
		}
	} else if authType == 2 && len(groupIds) > 0 {
		if err := models.DB.Delete(&models.OpsUserInstanceAuth{}, "user_id = ? and auth_type = ? and group_id in (?)", userId, authType, groupIds).Error; err != nil {
			log.Println("删除用户-主机分组授权关系异常: ", err)
			return errors.New("删除用户-主机分组授权关系异常")
		}
	}
	return nil
}

func (page *PageUserInstanceAuth) GetUserInstancesPage() (map[string]any, error) {
	result := make(map[string]any)

	userId := page.UserId
	if userId == 0 {
		return result, errors.New("用户ID不能为空")
	}

	var instances []models.OpsInstance
	if err := models.DB.Table("ops_instance").Select("ops_instance.*").Joins("JOIN ops_user_instance_auth ON ops_instance.id = ops_user_instance_auth.instance_id").Where("ops_user_instance_auth.user_id = ? AND ops_user_instance_auth.auth_type = 1", userId).Find(&instances).Error; err != nil {
		log.Println("获取用户授权主机信息异常: ", err)
		return result, errors.New("获取用户授权主机信息异常")
	}

	pageNum := page.PageNum
	pageSize := page.PageSize
	// 对instances 进行分页
	start := (pageNum - 1) * pageSize
	end := pageNum * pageSize
	if start >= len(instances) {
		return result, nil
	}
	if end > len(instances) {
		end = len(instances)
	}
	totalPage := int(math.Ceil(float64(len(instances)) / float64(pageSize)))
	//将分页后的结果存入result中
	result = map[string]any{
		"instances": instances[start:end],
		"total":     len(instances),
		"totalPage": totalPage,
		"pageNum":   pageNum,
		"pageSize":  pageSize,
	}
	return result, nil
}

func (page *PageUserInstanceAuth) GetUserGroupsPage() (map[string]any, error) {
	result := make(map[string]any)

	userId := page.UserId
	if userId == 0 {
		return result, errors.New("用户ID不能为空")
	}

	var groups []models.OpsGroup
	if err := models.DB.Table("ops_group").Select("ops_group.*").Joins("JOIN ops_user_instance_auth ON ops_group.id = ops_user_instance_auth.group_id").Where("ops_user_instance_auth.user_id = ? AND ops_user_instance_auth.auth_type = 2", userId).Find(&groups).Error; err != nil {
		log.Println("获取用户授权主机分组信息异常: ", err)
		return result, errors.New("获取用户授权主机分组信息异常")
	}

	pageNum := page.PageNum
	pageSize := page.PageSize
	// 对groups 进行分页
	start := (pageNum - 1) * pageSize
	end := pageNum * pageSize
	if start >= len(groups) {
		return result, nil
	}
	if end > len(groups) {
		end = len(groups)
	}
	totalPage := int(math.Ceil(float64(len(groups)) / float64(pageSize)))
	//将分页后的结果存入result中
	result = map[string]any{
		"groups":    groups[start:end],
		"total":     len(groups),
		"totalPage": totalPage,
		"pageNum":   pageNum,
		"pageSize":  pageSize,
	}

	return result, nil
}

// GetInstances 获取可绑定主机列表
func (page *PageUserInstanceAuth) GetInstances() (map[string]any, error) {
	var result = make(map[string]any)

	userId := page.UserId
	pageSize := page.PageSize
	pageNum := page.PageNum
	offset := (pageNum - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	// 获取用户已授权的主机
	var hasAuthedInstanceIds []int
	if err := models.DB.Model(&models.OpsUserInstanceAuth{}).Where("user_id = ? and auth_type = 1", userId).Select("instance_id").Find(&hasAuthedInstanceIds).Error; err != nil {
		log.Println("获取用户已授权主机异常: ", err)
		return result, errors.New("获取用户已授权主机异常")
	}

	var total int64
	var instances []models.OpsInstance
	if len(hasAuthedInstanceIds) == 0 {
		if err := models.DB.Model(&models.OpsInstance{}).Where("del_flag = ?", "0").Offset(offset).Limit(pageSize).Find(&instances).Error; err != nil {
			log.Println("获取主机信息异常: ", err)
			return result, errors.New("获取主机信息异常")
		}
		// 获取总条数
		models.DB.Model(&models.OpsInstance{}).Where("del_flag = ?", "0").Count(&total)
	} else {
		if err := models.DB.Table("ops_instance").Select("ops_instance.*").Where("ops_instance.del_flag = ? and ops_instance.id not in (?)", "0", hasAuthedInstanceIds).Offset(offset).Limit(pageSize).Find(&instances).Error; err != nil {
			log.Println("获取主机信息异常: ", err)
			return result, errors.New("获取主机信息异常")
		}
		// 获取总条数
		models.DB.Table("ops_instance").Select("ops_instance.id").Where("ops_instance.del_flag = ? and ops_instance.id not in (?)", "0", hasAuthedInstanceIds).Count(&total)
	}
	result["instances"] = instances
	result["total"] = total
	result["pageNum"] = pageNum
	result["pageSize"] = pageSize
	return result, nil
}

func (page *PageUserInstanceAuth) GetGroups() (map[string]any, error) {
	var result = make(map[string]any)

	userId := page.UserId
	pageSize := page.PageSize
	pageNum := page.PageNum
	offset := (pageNum - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	// 获取用户已授权的主机分组
	var hasAuthedGroupIds []int
	if err := models.DB.Model(&models.OpsUserInstanceAuth{}).Where("user_id = ? and auth_type = 2", userId).Select("group_id").Find(&hasAuthedGroupIds).Error; err != nil {
		log.Println("获取用户已授权主机分组异常: ", err)
		return result, errors.New("获取用户已授权主机分组异常")
	}
	var total int64
	var groups []models.OpsGroup
	if len(hasAuthedGroupIds) == 0 {
		if err := models.DB.Model(&models.OpsGroup{}).Where("del_flag = ?", "0").Offset(offset).Limit(pageSize).Find(&groups).Error; err != nil {
			log.Println("获取主机分组信息异常: ", err)
			return result, errors.New("获取主机分组信息异常")
		}
		// 获取总条数
		models.DB.Model(&models.OpsGroup{}).Where("del_flag = ?", "0").Count(&total)
	} else {
		if err := models.DB.Table("ops_group").Select("ops_group.*").Where("ops_group.del_flag = ? and ops_group.id not in (?)", "0", hasAuthedGroupIds).Offset(offset).Limit(pageSize).Find(&groups).Error; err != nil {
			log.Println("获取主机分组信息异常: ", err)
			return result, errors.New("获取主机分组信息异常")
		}
		// 获取总条数
		models.DB.Table("ops_group").Select("ops_group.id").Where("ops_group.del_flag = ? and ops_group.id not in (?)", "0", hasAuthedGroupIds).Count(&total)
	}

	result["groups"] = groups
	result["total"] = total
	result["pageNum"] = pageNum
	result["pageSize"] = pageSize
	return result, nil
}
