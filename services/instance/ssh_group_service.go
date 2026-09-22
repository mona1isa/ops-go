package instance

import (
	"errors"
	"github.com/zhany/ops-go/models"
	"log"
)

// MyGroupTree 查询当前用户有权限访问的主机分组树
type MyGroupTree struct {
	UserId  int  `json:"userId"`
	IsAdmin bool `json:"isAdmin"`
}

// SshKey 主机可用的登录凭证（不包含凭证密文）
type SshKey struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Type     int    `json:"type"`
}

// SshInstance 分组下的主机信息
type SshInstance struct {
	Id           int      `json:"id"`
	Name         string   `json:"name"`
	Ip           string   `json:"ip"`
	Os           string   `json:"os"`
	Spec         string   `json:"spec"`
	Status       string   `json:"status"`
	OnlineStatus string   `json:"onlineStatus"`
	Keys         []SshKey `json:"keys"`
}

// SshGroup 主机分组（包含分组下的主机）
type SshGroup struct {
	Id        int           `json:"id"`
	Name      string        `json:"name"`
	ParentId  int           `json:"parentId"`
	Children  []*SshGroup   `json:"children"`
	Instances []SshInstance `json:"instances"`
}

// UngroupedGroupId 未分组主机使用的虚拟分组ID
const UngroupedGroupId = -1

// GetMyGroupTree 获取当前用户有权限的主机分组树，包含分组下的主机与主机可用的登录凭证
// 超级管理员返回所有分组及主机绑定的凭证；普通用户返回已授权的分组、分组下的主机以及被授权的凭证
func (my *MyGroupTree) GetMyGroupTree() ([]*SshGroup, error) {
	userId := my.UserId
	isAdmin := my.IsAdmin

	// 查询全部可用凭证
	var keys []models.OpsKey
	if err := models.DB.Where("del_flag = ?", 0).Find(&keys).Error; err != nil {
		log.Println("查询登录凭证信息失败：", err)
		return nil, errors.New("查询登录凭证信息失败")
	}
	keyMap := make(map[int]models.OpsKey, len(keys))
	for _, key := range keys {
		keyMap[key.ID] = key
	}

	// 查询用户可见的主机分组
	var groups []models.OpsGroup
	if isAdmin {
		if err := models.DB.Where("del_flag = ?", 0).Order("id asc").Find(&groups).Error; err != nil {
			log.Println("查询主机分组信息失败：", err)
			return nil, errors.New("查询主机分组信息失败")
		}
	} else {
		if err := models.DB.Table("ops_group").Select("ops_group.*").
			Joins("JOIN ops_user_instance_auth ON ops_group.id = ops_user_instance_auth.group_id").
			Where("ops_user_instance_auth.user_id = ? AND ops_user_instance_auth.auth_type = 2 AND ops_user_instance_auth.del_flag = 0", userId).
			Order("ops_group.id asc").Find(&groups).Error; err != nil {
			log.Println("查询用户有权限的主机分组失败：", err)
			return nil, errors.New("查询用户有权限的主机分组失败")
		}
	}
	groupIds := make([]int, 0, len(groups))
	for _, group := range groups {
		groupIds = append(groupIds, group.ID)
	}

	// 查询用户可见的主机（普通用户只包含直接授权的主机与已授权分组下的主机）
	instanceDb := models.DB.Where("status = ? AND del_flag = ?", "1", 0)
	if !isAdmin {
		var auths []models.OpsUserInstanceAuth
		if err := models.DB.Where("user_id = ? AND del_flag = 0", userId).Find(&auths).Error; err != nil {
			log.Println("查询用户主机授权信息失败：", err)
			return nil, errors.New("查询用户主机授权信息失败")
		}
		var authInstanceIds []int
		var authGroupIds []int
		for _, auth := range auths {
			if auth.AuthType == 1 && auth.InstanceId != 0 {
				authInstanceIds = append(authInstanceIds, auth.InstanceId)
			}
			if auth.AuthType == 2 && auth.GroupId != 0 {
				authGroupIds = append(authGroupIds, auth.GroupId)
			}
		}
		if len(authGroupIds) > 0 {
			var groupInstanceIds []int
			if err := models.DB.Model(&models.OpsInstanceGroup{}).Where("group_id IN (?)", authGroupIds).Pluck("instance_id", &groupInstanceIds).Error; err != nil {
				log.Println("查询分组关联的主机失败：", err)
				return nil, errors.New("查询分组关联的主机失败")
			}
			authInstanceIds = append(authInstanceIds, groupInstanceIds...)
		}
		if len(authInstanceIds) == 0 {
			return []*SshGroup{}, nil
		}
		instanceDb = instanceDb.Where("id IN (?)", authInstanceIds)
	}

	var instances []models.OpsInstance
	if err := instanceDb.Order("id asc").Find(&instances).Error; err != nil {
		log.Println("查询主机信息失败：", err)
		return nil, errors.New("查询主机信息失败")
	}
	if len(instances) == 0 {
		return []*SshGroup{}, nil
	}

	instanceMap := make(map[int]models.OpsInstance, len(instances))
	instanceIds := make([]int, 0, len(instances))
	for _, instance := range instances {
		instanceMap[instance.ID] = instance
		instanceIds = append(instanceIds, instance.ID)
	}

	// 查询分组与主机的关联关系
	groupInstances := make(map[int][]int)
	if len(groupIds) > 0 {
		var relations []models.OpsInstanceGroup
		if err := models.DB.Where("group_id IN (?)", groupIds).Find(&relations).Error; err != nil {
			log.Println("查询分组与主机关联关系失败：", err)
			return nil, errors.New("查询分组与主机关联关系失败")
		}
		for _, relation := range relations {
			if _, ok := instanceMap[relation.InstanceId]; ok {
				groupInstances[relation.GroupId] = append(groupInstances[relation.GroupId], relation.InstanceId)
			}
		}
	}

	// 查询每个主机可用的凭证ID
	instanceBoundKeys := make(map[int][]int) // 管理员：主机绑定的全部凭证
	hostAuthKeys := make(map[int][]int)      // 普通用户：主机级授权的凭证
	groupAuthKeys := make(map[int][]int)     // 普通用户：分组级授权的凭证
	if isAdmin {
		var binds []models.OpsInstanceKey
		if err := models.DB.Where("instance_id IN (?)", instanceIds).Find(&binds).Error; err != nil {
			log.Println("查询主机绑定凭证失败：", err)
			return nil, errors.New("查询主机绑定凭证失败")
		}
		for _, bind := range binds {
			instanceBoundKeys[bind.InstanceId] = append(instanceBoundKeys[bind.InstanceId], bind.KeyId)
		}
	} else {
		var hostAuths []models.OpsUserInstanceKeyAuth
		if err := models.DB.Where("user_id = ? AND instance_id IN (?) AND auth_type = 1 AND del_flag = 0", userId, instanceIds).Find(&hostAuths).Error; err != nil {
			log.Println("查询用户主机凭证授权失败：", err)
			return nil, errors.New("查询用户主机凭证授权失败")
		}
		for _, auth := range hostAuths {
			hostAuthKeys[auth.InstanceId] = append(hostAuthKeys[auth.InstanceId], auth.KeyId)
		}
		if len(groupIds) > 0 {
			var groupAuths []models.OpsUserInstanceKeyAuth
			if err := models.DB.Where("user_id = ? AND group_id IN (?) AND auth_type = 2 AND del_flag = 0", userId, groupIds).Find(&groupAuths).Error; err != nil {
				log.Println("查询用户分组凭证授权失败：", err)
				return nil, errors.New("查询用户分组凭证授权失败")
			}
			for _, auth := range groupAuths {
				groupAuthKeys[auth.GroupId] = append(groupAuthKeys[auth.GroupId], auth.KeyId)
			}
		}
	}

	// 组装主机信息与可用凭证
	buildInstance := func(instanceId int, groupId int) SshInstance {
		instance := instanceMap[instanceId]
		item := SshInstance{
			Id:           instance.ID,
			Name:         instance.Name,
			Ip:           instance.Ip,
			Os:           instance.Os,
			Spec:         instance.Spec,
			Status:       instance.Status,
			OnlineStatus: instance.OnlineStatus,
			Keys:         []SshKey{},
		}
		var keyIds []int
		if isAdmin {
			keyIds = instanceBoundKeys[instanceId]
		} else {
			keyIds = append(keyIds, hostAuthKeys[instanceId]...)
			keyIds = append(keyIds, groupAuthKeys[groupId]...)
		}
		exist := make(map[int]bool, len(keyIds))
		for _, keyId := range keyIds {
			if exist[keyId] {
				continue
			}
			key, ok := keyMap[keyId]
			if !ok {
				continue
			}
			exist[keyId] = true
			item.Keys = append(item.Keys, SshKey{
				Id:       key.ID,
				Name:     key.Name,
				User:     key.User,
				Protocol: key.Protocol,
				Port:     key.Port,
				Type:     key.Type,
			})
		}
		return item
	}

	// 构建分组节点
	nodes := make([]*SshGroup, 0, len(groups))
	nodeMap := make(map[int]*SshGroup, len(groups))
	for _, group := range groups {
		node := &SshGroup{Id: group.ID, Name: group.Name, ParentId: group.ParentId, Instances: []SshInstance{}}
		nodes = append(nodes, node)
		nodeMap[group.ID] = node
	}
	grouped := make(map[int]bool, len(instances))
	for _, node := range nodes {
		for _, instanceId := range groupInstances[node.Id] {
			grouped[instanceId] = true
			node.Instances = append(node.Instances, buildInstance(instanceId, node.Id))
		}
	}

	// 未分组主机（直接授权给用户的主机）放入虚拟分组，避免主机无法访问
	var ungrouped []SshInstance
	for _, instanceId := range instanceIds {
		if grouped[instanceId] {
			continue
		}
		ungrouped = append(ungrouped, buildInstance(instanceId, UngroupedGroupId))
	}

	// 组装树形结构，父分组不可见时该分组作为根节点展示
	var roots []*SshGroup
	for _, node := range nodes {
		if parent, ok := nodeMap[node.ParentId]; ok && node.ParentId != node.Id {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	if len(ungrouped) > 0 {
		roots = append(roots, &SshGroup{Id: UngroupedGroupId, Name: "未分组主机", Instances: ungrouped})
	}
	return roots, nil
}
