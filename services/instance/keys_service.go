package instance

import (
	"errors"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
	"log"
)

type KeysService struct {
}

// ListKeys 获取密钥列表
func (s *KeysService) ListKeys() (keys []models.OpsKey, err error) {
	if err := models.DB.Where("status = ? AND del_flag = ?", "1", "0").Find(&keys).Error; err != nil {
		log.Println("查询密钥失败：", err)
		return nil, errors.New("查询密钥失败")
	}
	return keys, nil
}

// AddKey 添加密钥
func (s *KeysService) AddKey(request api.AddKeysRequest) (err error) {
	name := request.Name
	var count int64
	// 检查密钥名称是否已存在
	if err := models.DB.Model(&models.OpsKey{}).Where("name = ?", name).Count(&count).Error; err != nil {
		log.Println("添加密钥失败：", err)
		return errors.New("添加密钥失败")
	}
	if count > 0 {
		return errors.New("密钥名称已存在")
	}

	key := models.OpsKey{
		Name:        name,
		User:        request.User,
		Credentials: request.Credentials,
		Status:      request.Status,
		Protocol:    request.Protocol,
		Port:        request.Port,
		Type:        request.Type,
	}
	key.CreateBy = request.CreateBy
	key.UpdateBy = request.UpdateBy
	key.Remark = request.Remark
	if err := models.DB.Create(&key).Error; err != nil {
		log.Println("添加密钥失败：", err)
		return errors.New("添加密钥失败")
	}

	return nil
}

// EditKey 编辑密钥
func (s *KeysService) EditKey(request api.UpdateKeysRequest) (err error) {
	id := request.Id
	name := request.Name

	// 检查密钥是否存在及名称是否重复
	var key models.OpsKey
	if err := models.DB.Where("id = ?", id).First(&key).Error; err != nil {
		log.Println("编辑密钥失败：密钥不存在", err)
		return errors.New("密钥不存在")
	}

	// 检查名称是否重复
	var count int64
	if _ = models.DB.Where("name = ? AND id != ?", name, id).Count(&count); count > 0 {
		log.Println("编辑密钥失败：密钥名称已存在")
		return errors.New("密钥名称已存在")
	}

	// 更新密钥信息s
	key.Name = name
	key.User = request.User
	key.Credentials = request.Credentials
	key.Status = request.Status
	key.Protocol = request.Protocol
	key.Port = request.Port
	key.Type = request.Type
	key.UpdateBy = request.UpdateBy
	key.Remark = request.Remark

	if err := models.DB.Save(&key).Error; err != nil {
		log.Println("编辑密钥失败：", err)
		return errors.New("编辑密钥失败")
	}

	return nil
}

// PageKey 分页查询密钥
func (s *KeysService) PageKey(request api.PageKeysRequest) (models.PageResult[models.OpsKey], error) {
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

	pageResult, err := models.Paginate[models.OpsKey](models.DB, pageNum, pageSize, scopes...)
	if err != nil {
		log.Println("查询主机列表异常：", err)
		panic(err)
	}

	return pageResult, nil
}

// ChangeStatus 修改密钥状态
func (s *KeysService) ChangeStatus(request api.ChangeStatusRequest) (err error) {
	id := request.Id
	status := request.Status
	if err := models.DB.Model(&models.OpsKey{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		log.Println("更新密钥状态失败：", err)
		return errors.New("更新密钥状态失败")
	}
	return nil
}

// DeleteKey 删除密钥
func (s *KeysService) DeleteKey(id int) (err error) {
	if err := models.DB.Delete(&models.OpsKey{}, id).Error; err != nil {
		return errors.New("删除主机失败")
	}
	return nil
}
