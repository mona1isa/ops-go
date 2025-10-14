package instance

import (
	"errors"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"log"
	"strconv"
)

type GroupService struct {
}

func (s *GroupService) AddGroup(request api.AddGroupRequest) (err error) {
	name := request.Name
	// 校验名称是否重复
	if isExist, _ := s.IsGroupExist(name, 0); isExist {
		return errors.New("分组名称已存在")
	}

	// 保存分组
	group := models.OpsGroup{
		Name: name,
	}
	parentIdStr := request.ParentId
	if parentIdStr != "" {
		parentId, _ := strconv.Atoi(parentIdStr)
		group.ParentId = parentId
	}

	group.CreateBy = request.CreateBy
	group.UpdateBy = request.UpdateBy
	if err = models.DB.Create(&group).Error; err != nil {
		log.Println("保存分组失败：", err)
		return errors.New("保存分组失败")
	}
	return nil
}

// EditGroup 编辑分组
func (s *GroupService) EditGroup(request api.UpdateGroupRequest) (err error) {
	id := request.Id
	name := request.Name

	var group models.OpsGroup
	if err = models.DB.Where("id = ?", id).First(&group).Error; err != nil {
		log.Println("查询分组失败：", err)
		return errors.New("查询分组失败")
	}

	// 校验名称是否重复
	if isExist, _ := s.IsGroupExist(name, id); isExist {
		return errors.New("分组名称已存在")
	}

	// 更新分组
	group.Name = name
	group.UpdateBy = request.UpdateBy
	if err = models.DB.Model(&models.OpsGroup{}).Where("id = ?", id).Updates(&group).Error; err != nil {
		log.Println("更新分组失败：", err)
		return errors.New("更新分组失败")
	}
	return nil
}

// ListGroup 查询分组树形结构
func (s *GroupService) ListGroup() ([]*models.OpsGroup, error) {
	var all []models.OpsGroup
	if err := models.DB.Find(&all).Error; err != nil {
		log.Println("查询分组失败：", err)
		return nil, errors.New("查询分组失败")
	}

	return buildTree(all), nil
}

// 将查询出来的分组数据组装成树形结构
func buildTree(groups []models.OpsGroup) []*models.OpsGroup {
	var root []*models.OpsGroup
	groupMap := make(map[int]*models.OpsGroup)

	// 初始化映射
	for i := range groups {
		groupMap[groups[i].ID] = &groups[i]
	}

	// 构建树形结构
	for i := range groups {
		if groups[i].ParentId == 0 {
			root = append(root, &groups[i])
		} else {
			if parent, ok := groupMap[groups[i].ParentId]; ok {
				if parent.Children == nil {
					parent.Children = []*models.OpsGroup{}
				}
				parent.Children = append(parent.Children, &groups[i])
			}
		}
	}

	return root
}

// DeleteGroup 删除分组
func (s *GroupService) DeleteGroup(id int) (err error) {
	// 校验是否存在子分组，如果有则不允许删除
	var count int64
	if err = models.DB.Model(&models.OpsGroup{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		log.Println("查询分组是否存在失败：", err)
		return errors.New("查询分组是否存在失败")
	}
	if count > 0 {
		return errors.New("存在子分组，不允许删除")
	}

	// 校验分组下是否存在实例，如果有则不允许删除
	var instanceCount int64
	if err := models.DB.Model(&models.OpsInstanceGroup{}).Where("group_id = ?", id).Count(&instanceCount).Error; err != nil {
		log.Println("查询分组是否存在实例失败：", err)
		return errors.New("查询分组是否存在实例失败")
	}
	if instanceCount > 0 {
		return errors.New("分组下存在实例，不允许删除")
	}

	// 校验分组是否存在实例，如果有则不允许删除
	if err = models.DB.Where("id = ?", id).Delete(&models.OpsGroup{}).Error; err != nil {
		log.Println("删除分组失败：", err)
		return errors.New("删除分组失败")
	}
	return nil
}

// IsGroupExist 校验分组是否存在
func (s *GroupService) IsGroupExist(name string, id int) (isExist bool, err error) {
	var count int64
	if id > 0 {
		if err = models.DB.Model(&models.OpsGroup{}).Where("id != ? and name = ?", id, name).Count(&count).Error; err != nil {
			log.Println("查询分组是否存在失败：", err)
			return
		}
	} else {
		if err = models.DB.Model(&models.OpsGroup{}).Where("name = ?", name).Count(&count).Error; err != nil {
			log.Println("查询分组是否存在失败：", err)
			return
		}
	}

	if count > 0 {
		isExist = true
	}
	return isExist, nil
}

// GroupInstanceOps 向分组添加/移除实例
func (s *GroupService) GroupInstanceOps(request api.GroupInstanceRequest) (err error) {
	groupId := request.GroupId
	instanceIds := request.InstanceIds
	opsType := request.OpsType

	// 校验分组是否存在
	var count int64
	if err = models.DB.Model(&models.OpsGroup{}).Where("id = ?", groupId).Count(&count).Error; err != nil {
		log.Println("查询分组是否存在失败：", err)
		return errors.New("查询分组是否存在失败")
	}
	if count == 0 {
		return errors.New("分组不存在")
	}

	if len(instanceIds) == 0 {
		return errors.New("请选择实例")
	}

	// 执行分组实例操作
	if opsType == "add" {
		var instanceGroup []models.OpsInstanceGroup
		for _, instanceId := range instanceIds {
			instanceGroup = append(instanceGroup, models.OpsInstanceGroup{
				GroupId:    groupId,
				InstanceId: instanceId,
			})
		}
		if err = models.DB.Create(&instanceGroup).Error; err != nil {
			log.Println("添加实例到分组失败：", err)
			return errors.New("添加实例到分组失败")
		}
	} else if opsType == "remove" {
		if err = models.DB.Where("group_id = ? and instance_id in (?)", groupId, instanceIds).Delete(&models.OpsInstanceGroup{}).Error; err != nil {
			log.Println("移除实例到分组失败：", err)
			return errors.New("移除实例到分组失败")
		}
	} else {
		return errors.New("操作类型错误")
	}
	return nil
}

// PageGroupInstance 分页查询分组实例
func (s *GroupService) PageGroupInstance(request api.PageGroupInstanceRequest) (page api.PageGroupInstanceResponse, err error) {
	// 校验分组是否存在
	var count int64
	if err = models.DB.Model(&models.OpsGroup{}).Where("id = ?", request.GroupId).Count(&count).Error; err != nil {
		log.Println("查询分组是否存在失败：", err)
		return page, errors.New("查询分组是否存在失败")
	}
	if count == 0 {
		return page, errors.New("分组不存在")
	}
	// 分页查询分组下实例
	// SELECT * FROM ops_instance WHERE id IN (SELECT instance_id FROM ops_instance_group WHERE group_id = ?) OFFSET pageNum*pageSize LIMIT pageSize
	var instances []models.OpsInstance
	if err = models.DB.Table("ops_instance").Select("ops_instance.*").Joins("JOIN ops_instance_group ON ops_instance.id = ops_instance_group.instance_id").Where("ops_instance_group.group_id = ?", request.GroupId).Offset((request.PageNum - 1) * request.PageSize).Limit(request.PageSize).Find(&instances).Error; err != nil {
		log.Println("查询分组实例失败：", err)
		return page, errors.New("查询分组实例失败")
	}
	// 查询分组实例总数
	if err = models.DB.Table("ops_instance").Joins("JOIN ops_instance_group ON ops_instance.id = ops_instance_group.instance_id").Where("ops_instance_group.group_id = ?", request.GroupId).Count(&page.Total).Error; err != nil {
		log.Println("查询分组实例总数失败：", err)
		return page, errors.New("查询分组实例总数失败")
	}
	page.Data = instances
	page.PageNum = request.PageNum
	page.PageSize = request.PageSize
	return page, nil
}

// AvailableInstance 查询不在分组内的实例用于添加到分组中
func (s *GroupService) AvailableInstance(request api.PageGroupInstanceRequest) (page api.PageGroupInstanceResponse, err error) {
	// 校验分组是否存在
	var count int64
	if err = models.DB.Model(&models.OpsGroup{}).Where("id = ?", request.GroupId).Count(&count).Error; err != nil {
		log.Println("查询分组是否存在失败：", err)
		return page, errors.New("查询分组是否存在失败")
	}
	if count == 0 {
		return page, errors.New("分组不存在")
	}

	// 查询分组内实例ID列表
	var instanceIds []string
	if err = models.DB.Table("ops_instance_group").Select("instance_id").Where("group_id = ?", request.GroupId).Find(&instanceIds).Error; err != nil {
		log.Println("查询分组实例列表失败：", err)
		return page, errors.New("查询分组实例列表失败")
	}

	// 分页查询不在分组内的实例
	var instances []models.OpsInstance
	if len(instanceIds) == 0 {
		if err := models.DB.Table("ops_instance").Offset((request.PageNum - 1) * request.PageSize).Limit(request.PageSize).Find(&instances).Error; err != nil {
			log.Println("查询分组实例失败：", err)
			return page, errors.New("查询分组实例失败")
		}
		// 查询分组实例总数
		if err := models.DB.Table("ops_instance").Count(&page.Total).Error; err != nil {
			log.Println("查询分组实例总数失败：", err)
			return page, errors.New("查询分组实例总数失败")
		}
	} else {
		if err := models.DB.Table("ops_instance").Where("id NOT IN (?)", instanceIds).Offset((request.PageNum - 1) * request.PageSize).Limit(request.PageSize).Find(&instances).Error; err != nil {
			log.Println("查询分组实例失败：", err)
			return page, errors.New("查询分组实例失败")
		}
		// 查询分组实例总数
		if err := models.DB.Table("ops_instance").Where("id NOT IN (?)", instanceIds).Count(&page.Total).Error; err != nil {
			log.Println("查询分组实例总数失败：", err)
			return page, errors.New("查询分组实例总数失败")
		}
	}

	page.Data = instances
	page.PageNum = request.PageNum
	page.PageSize = request.PageSize
	return page, nil
}
