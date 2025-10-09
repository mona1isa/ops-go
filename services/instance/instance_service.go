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
	deptId := request.DeptId
	name := request.Name
	var count int64
	if err := models.DB.Model(&models.OpsInstance{}).Where("dept_id = ? and name = ?", deptId, name).Count(&count).Error; err != nil {
		return errors.New("添加主机失败")
	}
	if count > 0 {
		return errors.New("主机名称已存在")
	}

	instance := models.OpsInstance{
		DeptId: deptId,
		Name:   name,
		Cpu:    request.Cpu,
		Mem:    request.Mem,
		Disk:   request.Disk,
		Ip:     request.Ip,
		Port:   request.Port,
		Os:     request.Os,
		Status: request.Status,
	}

	instance.CreateBy = request.CreateBy
	instance.UpdateBy = request.UpdateBy
	if err = models.DB.Save(&instance).Error; err != nil {
		return errors.New("添加主机失败")
	}
	return
}

// EditInstance 编辑实例
func (s *InstanceService) EditInstance(request api.UpdateInstanceRequest) (err error) {
	id := request.Id
	deptId := request.DeptId
	name := request.Name
	var count int64
	if err := models.DB.Model(&models.OpsInstance{}).Where("dept_id = ? and name = ? and id != ?", deptId, name, id).Count(&count).Error; err != nil {
		return errors.New("添加主机失败")
	}
	if count > 0 {
		return errors.New("主机名称已存在")
	}

	var instance models.OpsInstance
	models.DB.First(&instance, id)
	instance.DeptId = deptId
	instance.Name = name
	instance.Cpu = request.Cpu
	instance.Mem = request.Mem
	instance.Disk = request.Disk
	instance.Ip = request.Ip
	instance.Port = request.Port
	instance.Os = request.Os
	instance.Status = request.Status
	instance.UpdateBy = request.UpdateBy
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
	if request.DeptId != 0 {
		scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
			return db.Where("dept_id = ?", request.DeptId)
		})
	}

	pageResult, err := models.Paginate[models.OpsInstance](models.DB, pageNum, pageSize, scopes...)
	if err != nil {
		log.Println("查询主机列表异常：", err)
		panic(err)
	}

	return pageResult, nil
}

// GetInstanceDetail 获取实例详细信息
func (s *InstanceService) GetInstanceDetail(id int) (instance models.OpsInstance, err error) {
	models.DB.First(&instance, id)
	if instance.ID == 0 {
		return instance, errors.New("主机不存在")
	}
	return
}

// DeleteInstance 删除实例
func (s *InstanceService) DeleteInstance(id int) (err error) {
	if err := models.DB.Delete(&models.OpsInstance{}, id).Error; err != nil {
		return errors.New("删除主机失败")
	}
	return nil
}
