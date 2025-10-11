package instance

import (
	"errors"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
	"log"
)

type InstanceService struct{}

// AddInstance 添加实例
func (s *InstanceService) AddInstance(request api.AddInstanceRequest) (err error) {
	name := request.Name
	var count int64
	if err := models.DB.Model(&models.OpsInstance{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return errors.New("添加主机失败")
	}
	if count > 0 {
		return errors.New("主机名称已存在")
	}

	instance := models.OpsInstance{
		Name:   name,
		Cpu:    request.Cpu,
		MemMb:  request.MemMb,
		DiskGb: request.DiskGb,
		Ip:     request.Ip,
		Os:     request.Os,
		Status: request.Status,
	}

	instance.CreateBy = request.CreateBy
	instance.UpdateBy = request.UpdateBy
	instance.Remark = request.Remark
	if err = models.DB.Save(&instance).Error; err != nil {
		log.Println("添加主机失败：", err)
		return errors.New("添加主机失败")
	}
	return
}

// EditInstance 编辑实例
func (s *InstanceService) EditInstance(request api.UpdateInstanceRequest) (err error) {
	id := request.Id
	name := request.Name
	var count int64
	if err := models.DB.Model(&models.OpsInstance{}).Where("name = ? and id != ?", name, id).Count(&count).Error; err != nil {
		return errors.New("添加主机失败")
	}
	if count > 0 {
		return errors.New("主机名称已存在")
	}

	var instance models.OpsInstance
	models.DB.First(&instance, id)

	instance.Name = name
	instance.Cpu = request.Cpu
	instance.MemMb = request.MemMb
	instance.DiskGb = request.DiskGb
	instance.Ip = request.Ip
	instance.Os = request.Os
	instance.Status = request.Status
	instance.UpdateBy = request.UpdateBy
	instance.Remark = request.Remark
	if err := models.DB.Save(&instance).Error; err != nil {
		return errors.New("编辑主机失败")
	}

	return nil
}

// ChangeStatus 修改实例状态
func (s *InstanceService) ChangeStatus(request api.ChangeStatusRequest) (err error) {
	id := request.Id
	status := request.Status
	if err := models.DB.Model(&models.OpsInstance{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return errors.New("修改主机状态失败")
	}
	return nil
}

// PageInstance 分页查询实例
func (s *InstanceService) PageInstance(request api.PageInstanceRequest) (models.PageResult[models.OpsInstance], error) {
	pageNum := request.PageNum
	pageSize := request.PageSize

	var scopes []func(db *gorm.DB) *gorm.DB
	if request.Name != "" {
		scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
			return db.Where("name like ?", "%"+request.Name+"%")
		})
	}
	if request.Status != "" {
		scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", request.Status)
		})
	}

	pageResult, err := models.Paginate[models.OpsInstance](models.DB, pageNum, pageSize, scopes...)
	if err != nil {
		log.Println("查询主机列表异常：", err)
		panic(err)
	}

	// 查询主机绑定的密钥
	for _, instance := range pageResult.Data {
		id := instance.ID
		// select * from ops_key where id in(select key_id from ops_instance_keys where instance_id =?)
		var instanceKeys []models.OpsInstanceKey
		if err := models.DB.Table("ops_key").Select("id, name").Joins("join ops_instance_keys on ops_key.id = ops_instance_keys.key_id").Where("ops_instance_keys.instance_id = ?", id).Find(&instanceKeys).Error; err != nil {
			return pageResult, errors.New("查询主机列表失败")
		}
		instance.BindingKeys = instanceKeys
	}

	return pageResult, nil
}

// GetInstanceDetail 获取实例详细信息
func (s *InstanceService) GetInstanceDetail(id int) (instance models.OpsInstance, err error) {
	models.DB.First(&instance, id)
	if instance.ID == 0 {
		return instance, errors.New("主机不存在")
	}
	// 查询实例-凭证关系
	var instanceKeys []models.OpsInstanceKey
	// select id, name from ops_key where id in (select key_id from ops_instance_key where instance_id = ?)
	if err := models.DB.Table("ops_key").Select("id, name").Joins("join ops_instance_key on ops_key.id = ops_instance_key.key_id").Where("ops_instance_key.instance_id = ?", id).Find(&instanceKeys).Error; err != nil {
		return instance, errors.New("查询主机详情失败")
	}
	instance.BindingKeys = instanceKeys
	return
}

// DeleteInstance 删除实例
func (s *InstanceService) DeleteInstance(id int) (err error) {
	if err := models.DB.Delete(&models.OpsInstance{}, id).Error; err != nil {
		return errors.New("删除主机失败")
	}
	return nil
}

// KeyBinding 主机绑定密钥
func (s *InstanceService) KeyBinding(request api.InstanceKeyBindingRequest) (err error) {
	instanceId := request.InstanceId
	keyId := request.KeyId

	// 检查实例是否存在
	var instance models.OpsInstance
	if err := models.DB.First(&instance, instanceId).Error; err != nil {
		log.Println("绑定密钥失败：", err)
		return errors.New("实例不存在, 绑定密钥失败")
	}

	// 检查密钥是否存在
	var key models.OpsKey
	if err := models.DB.First(&key, keyId).Error; err != nil {
		log.Println("绑定密钥失败：", err)
		return errors.New("密钥不存在, 绑定密钥失败")
	}

	// 保存主机-凭证关系
	if err := models.DB.Create(&models.OpsInstanceKey{InstanceId: instanceId, KeyId: keyId}).Error; err != nil {
		log.Println("绑定密钥失败：", err)
		return errors.New("绑定密钥失败")
	}

	return nil
}
