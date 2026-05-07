package instance

import (
	"errors"
	"github.com/zhany/ops-go/models"
	"log"
)

type MyInstance struct {
	PageNum  int    `json:"pageNum"`
	PageSize int    `json:"pageSize"`
	UserId   int    `json:"userId"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isAdmin"`
}

type MyInstanceService interface {
	GetMyInstance(userId int) (any, error)
}

// GetMyInstance 获取用户有权限的主机信息
func (my *MyInstance) GetMyInstance() (map[string]any, error) {
	result := make(map[string]any)
	isAdmin := my.IsAdmin
	// 查询所有登录凭证信息
	var keys []models.OpsKey
	if err := models.DB.Where("del_flag = ?", 0).Find(&keys).Error; err != nil {
		return result, errors.New("查询主机登录凭证信息失败")
	}
	// 将凭证信息转换为map, key 为凭证id, value为凭证信息
	var keyMap = make(map[int]models.OpsKey)
	for _, key := range keys {
		keyMap[key.ID] = key
	}

	if isAdmin {
		// 分页查询所有的主机信息
		var pageInstances []*models.OpsInstance
		db := models.DB.Where("status = ? and del_flag = ?", 1, 0)
		if my.Name != "" {
			db = db.Where("name like ?", "%"+my.Name+"%")
		}
		if err := db.Offset((my.PageNum - 1) * my.PageSize).Limit(my.PageSize).Find(&pageInstances).Error; err != nil {
			return result, errors.New("查询主机信息失败")
		}
		// 查询总数
		var total int64
		countDb := models.DB.Model(&models.OpsInstance{}).Where("status = ? and del_flag = ?", 1, 0)
		if my.Name != "" {
			countDb = countDb.Where("name like ?", "%"+my.Name+"%")
		}
		if err := countDb.Count(&total).Error; err != nil {
			return result, errors.New("查询主机信息失败")
		}
		// 查询主机关联的凭证
		var instanceKeys []models.OpsInstanceKey
		if err := models.DB.Find(&instanceKeys).Error; err != nil {
			return result, errors.New("查询主机关联的凭证失败")
		}
		// 组装返回结果
		for _, instance := range pageInstances {
			for _, instanceKey := range instanceKeys {
				if instanceKey.InstanceId == instance.ID {
					instance.BindingKeys = append(instance.BindingKeys, keyMap[instanceKey.KeyId])
				}
			}
		}
		result["data"] = pageInstances
		result["total"] = total
		result["totalPage"] = int(total / int64(my.PageSize))
	} else {
		log.Println("查询用户有权限访问的主机信息，UserId=", my.UserId)
		var userInstanceAuths []models.OpsUserInstanceAuth
		if err := models.DB.Model(&models.OpsUserInstanceAuth{}).Where("user_id = ? AND del_flag=0", my.UserId).Find(&userInstanceAuths).Error; err != nil {
			return result, errors.New("查询用户有权限访问的主机信息失败")
		}
		// 分离出凭证id
		var groupIds []int
		// 分离出主机id
		var authInstanceIds []int
		for _, userInstanceAuth := range userInstanceAuths {
			if userInstanceAuth.AuthType == 1 && userInstanceAuth.InstanceId != 0 {
				authInstanceIds = append(authInstanceIds, userInstanceAuth.InstanceId)
			}
			if userInstanceAuth.AuthType == 2 && userInstanceAuth.GroupId != 0 {
				groupIds = append(groupIds, userInstanceAuth.GroupId)
			}
		}
		// 查询与分组绑定的所有主机ID
		var groupInstanceIds []int
		if len(groupIds) > 0 {
			if err := models.DB.Model(&models.OpsInstanceGroup{}).Select("instance_id").Where("group_id IN (?)", groupIds).Find(&groupInstanceIds).Error; err != nil {
				log.Println("查询分组关联的主机信息失败，Error=", err)
				return result, errors.New("查询分组关联的主机信息失败")
			}
			if len(groupInstanceIds) > 0 {
				authInstanceIds = append(authInstanceIds, groupInstanceIds...)
			}
		}

		// 分页查询主机信息
		var pageInstances []*models.OpsInstance
		db := models.DB.Where("id IN (?) AND status = ? AND del_flag = ?", authInstanceIds, 1, 0)
		if my.Name != "" {
			db = db.Where("name like ?", "%"+my.Name+"%")
		}
		if err := db.Offset((my.PageNum - 1) * my.PageSize).Limit(my.PageSize).Find(&pageInstances).Error; err != nil {
			log.Println("查询主机信息失败，Error=", err)
			return result, errors.New("查询主机信息失败")
		}
		// 查询总数
		var total int64
		countDb := models.DB.Model(&models.OpsInstance{}).Where("id IN (?) AND status = ? AND del_flag = ?", authInstanceIds, 1, 0)
		if my.Name != "" {
			countDb = countDb.Where("name like ?", "%"+my.Name+"%")
		}
		if err := countDb.Count(&total).Error; err != nil {
			log.Println("查询主机信息失败，Error=", err)
			return result, errors.New("查询主机信息失败")
		}

		// 查询用户-主机关联的凭证
		var userInstanceKeyAuths []models.OpsUserInstanceKeyAuth
		if len(authInstanceIds) > 0 {
			if err := models.DB.Model(&models.OpsUserInstanceKeyAuth{}).Where("user_id = ? AND instance_id IN (?) AND del_flag=0", my.UserId, authInstanceIds).Find(&userInstanceKeyAuths).Error; err != nil {
				log.Println("查询用户-主机关联的凭证失败，Error=", err)
				return result, errors.New("查询用户-主机关联的凭证失败")
			}

			// 设置主机关联的凭证信息
			if len(userInstanceKeyAuths) > 0 {
				for _, instance := range pageInstances {
					for _, instanceKeyAuth := range userInstanceKeyAuths {
						if instance.ID == instanceKeyAuth.InstanceId {
							instance.BindingKeys = append(instance.BindingKeys, keyMap[instanceKeyAuth.KeyId])
						}
					}
				}
			}
		}

		// 查询用户-分组授权的凭证
		var userGroupKeyAuths []models.OpsUserInstanceKeyAuth
		if len(groupIds) > 0 {
			if err := models.DB.Model(&models.OpsUserInstanceKeyAuth{}).Where("user_id = ? AND group_id IN (?) AND del_flag=0", my.UserId, groupIds).Find(&userGroupKeyAuths).Error; err != nil {
				log.Println("查询用户-分组授权的凭证失败，Error=", err)
				return result, errors.New("查询用户-分组授权的凭证失败")
			}

			var instanceKeyMap = make(map[int][]int)
			if len(userGroupKeyAuths) > 0 {
				var instanceGroups []models.OpsInstanceGroup
				if err := models.DB.Model(&models.OpsInstanceGroup{}).Select("group_id, instance_id").Where("group_id IN (?) AND del_flag=0", groupIds).Find(&instanceGroups).Error; err != nil {
					log.Println("查询分组关联的主机信息失败，Error=", err)
					return result, errors.New("查询分组关联的主机信息失败")
				}

				if len(instanceGroups) > 0 {
					for _, instanceGroup := range instanceGroups {
						for _, userGroupKeyAuth := range userGroupKeyAuths {
							if instanceGroup.GroupId == userGroupKeyAuth.GroupId {
								instanceKeyMap[instanceGroup.InstanceId] = append(instanceKeyMap[instanceGroup.InstanceId], userGroupKeyAuth.KeyId)
							}
						}
					}
				}
			}
			// 设置主机关联的凭证信息
			if len(instanceKeyMap) > 0 {
				for _, instance := range pageInstances {
					for _, keyId := range instanceKeyMap[instance.ID] {
						instance.BindingKeys = append(instance.BindingKeys, keyMap[keyId])
					}
				}
			}
		}

		result["data"] = pageInstances
		result["total"] = total
		result["totalPage"] = int(total / int64(my.PageSize))
	}
	return result, nil
}
