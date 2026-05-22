package instance

import (
	"errors"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/utils"
	"gorm.io/gorm"
	"log"
	"strings"
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

	// 加密存储
	credentials := request.Credentials
	encrypted, err := utils.EncryptPassword(credentials)
	if err != nil {
		log.Println("加密凭证失败：", err)
		return errors.New("加密凭证失败")
	}
	credentials = encrypted

	key := models.OpsKey{
		Name:        name,
		User:        request.User,
		Credentials: credentials,
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
	key.Status = request.Status
	key.Protocol = request.Protocol
	key.Port = request.Port
	key.Type = request.Type
	key.UpdateBy = request.UpdateBy
	key.Remark = request.Remark

	// 如果是密码或密钥类型且凭证有变化，则加密存储
	// 防护：如果前端传回的是已加密的密文（解密后与原值不同），则不再二次加密
	if (request.Type == 1 || request.Type == 2) && request.Credentials != "" {
		decrypted, _ := utils.DecryptKey(request.Credentials)
		if decrypted == request.Credentials {
			// 传入的是明文，需要加密
			encrypted, err := utils.EncryptPassword(request.Credentials)
			if err != nil {
				log.Println("加密凭证失败：", err)
				return errors.New("加密凭证失败")
			}
			key.Credentials = encrypted
		}
		// 否则传入的是密文，保留数据库原值，不做更新
	}

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

// AvailableKeys 获取实例可用凭证
func (s *KeysService) AvailableKeys(instanceId int) (keys []models.OpsKey, err error) {
	// 获取实例信息
	var instance models.OpsInstance
	if err := models.DB.First(&instance, instanceId).Error; err != nil {
		return nil, errors.New("获取实例信息失败")
	}

	protocol := "ssh"
	if strings.EqualFold(instance.Os, "windows") {
		protocol = "rdp"
	}
	// 获取所有凭证
	var allKeys []models.OpsKey
	if err := models.DB.Where("protocol = ? AND status = ? AND del_flag = ?", protocol, "1", "0").Find(&allKeys).Error; err != nil {
		log.Println("查询密钥失败：", err)
		return nil, errors.New("查询密钥失败")
	}

	// 获取实例已绑定的凭证
	var instanceKeys []models.OpsInstanceKey
	if err := models.DB.Where("instance_id = ?", instanceId).Find(&instanceKeys).Error; err != nil {
		log.Println("查询实例密钥失败：", err)
		return nil, errors.New("查询实例密钥失败")
	}

	// 从所有凭证中过滤出未绑定的凭证
	for _, key := range allKeys {
		isBind := false
		for _, instanceKey := range instanceKeys {
			if key.ID == instanceKey.KeyId {
				isBind = true
				break
			}
		}
		if !isBind {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// AvailableKeysBySystem 获取系统可用凭证（创建主机时使用）
func (s *KeysService) AvailableKeysBySystem(request api.OsTypeRequest) (keys []models.OpsKey, err error) {
	osType := request.OsType
	if osType == "" {
		if err := models.DB.Where("status = ? AND del_flag = ?", "1", "0").Find(&keys).Error; err != nil {
			return nil, errors.New("查询密钥失败")
		}
		return keys, nil
	}
	protocol := "ssh"
	if strings.EqualFold(osType, "windows") {
		protocol = "rdp"
	}
	// 获取所有凭证
	if err := models.DB.Where("protocol = ? AND status = ? AND del_flag = ?", protocol, "1", "0").Find(&keys).Error; err != nil {
		log.Println("查询密钥失败：", err)
		return nil, errors.New("查询密钥失败")
	}
	return keys, nil
}

// GetKeyDetail 获取凭证详情
func (s *KeysService) GetKeyDetail(id int) (models.OpsKey, error) {
	var key models.OpsKey
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&key).Error; err != nil {
		log.Println("查询凭证详情失败：", err)
		return key, errors.New("凭证不存在")
	}
	return key, nil
}

// GetKeyInstances 获取凭证绑定的主机列表（分页）
func (s *KeysService) GetKeyInstances(keyId, pageNum, pageSize int) (models.PageResult[models.OpsInstance], error) {
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("id IN (?)",
			models.DB.Table("ops_instance_keys").Select("instance_id").Where("key_id = ?", keyId),
		).Where("del_flag = ?", "0")
	}
	result, err := models.Paginate[models.OpsInstance](models.DB, pageNum, pageSize, scope)
	if err != nil {
		log.Println("查询凭证绑定的主机列表异常：", err)
		return result, errors.New("查询绑定的主机列表失败")
	}
	return result, nil
}

// GetAvailableInstances 获取未绑定该凭证的主机列表
func (s *KeysService) GetAvailableInstances(keyId int, name, ip string) ([]models.OpsInstance, error) {
	db := models.DB.Where("del_flag = ? AND status = ?", "0", "1").
		Where("id NOT IN (?)",
			models.DB.Table("ops_instance_keys").Select("instance_id").Where("key_id = ?", keyId),
		)
	if name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	if ip != "" {
		db = db.Where("ip LIKE ?", "%"+ip+"%")
	}
	var instances []models.OpsInstance
	if err := db.Find(&instances).Error; err != nil {
		log.Println("查询可用主机列表异常：", err)
		return nil, errors.New("查询可用主机列表失败")
	}
	return instances, nil
}

// BindInstances 批量绑定主机到凭证
func (s *KeysService) BindInstances(keyId int, instanceIds []int) error {
	var key models.OpsKey
	if err := models.DB.First(&key, keyId).Error; err != nil {
		return errors.New("凭证不存在")
	}

	var alreadyBound int64
	models.DB.Model(&models.OpsInstanceKey{}).
		Where("key_id = ? AND instance_id IN ?", keyId, instanceIds).
		Count(&alreadyBound)
	if alreadyBound > 0 {
		return errors.New("部分主机已绑定该凭证，请刷新后重试")
	}

	records := make([]models.OpsInstanceKey, 0, len(instanceIds))
	for _, instanceId := range instanceIds {
		records = append(records, models.OpsInstanceKey{KeyId: keyId, InstanceId: instanceId})
	}
	if err := models.DB.Create(&records).Error; err != nil {
		log.Println("批量绑定主机失败：", err)
		return errors.New("批量绑定主机失败")
	}
	return nil
}

// UnbindInstance 解绑凭证下的某个主机（同时清除用户授权记录）
func (s *KeysService) UnbindInstance(keyId, instanceId int) error {
	result := models.DB.Where("key_id = ? AND instance_id = ?", keyId, instanceId).
		Delete(&models.OpsInstanceKey{})
	if result.Error != nil {
		log.Println("解绑主机失败：", result.Error)
		return errors.New("解绑主机失败")
	}
	if result.RowsAffected == 0 {
		return errors.New("绑定关系不存在")
	}

	// 同时清除该主机-凭证的所有用户授权记录，否则用户在"我的主机"和SSH连接中仍会看到该凭证
	if err := models.DB.Where("instance_id = ? AND key_id = ?", instanceId, keyId).
		Delete(&models.OpsUserInstanceKeyAuth{}).Error; err != nil {
		log.Println("清除用户凭证授权记录失败：", err)
	}
	return nil
}

// GetPublicKey 获取公钥用于加密凭证
func (s *KeysService) GetPublicKey() (string, error) {
	pubKey, err := utils.GetPublicKey()
	if err != nil {
		log.Println("获取公钥失败：", err)
		return "", errors.New("获取公钥失败")
	}
	publicKey, err := utils.ExportPublicKeyToPEM(pubKey)
	if err != nil {
		log.Println("导出公钥失败：", err)
		return "", errors.New("导出公钥失败")
	}
	return publicKey, nil
}
