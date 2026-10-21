package monitor

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/monitor/datasource"
	"gorm.io/gorm"
)

// TargetService 监控采集目标管理服务
type TargetService struct{}

// TargetFilter 列表查询条件
type TargetFilter struct {
	Name       string
	InstanceIp string
	LastStatus *int8
	Enabled    string
}

// List 分页查询采集目标
func (s *TargetService) List(pageNum, pageSize int, filter TargetFilter) (models.PageResult[models.OpsMonitorTarget], error) {
	result, err := models.Paginate[models.OpsMonitorTarget](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		db = db.Where("del_flag = ?", "0")
		if filter.Name != "" {
			db = db.Where("instance_name like ?", "%"+escapeLike(filter.Name)+"%")
		}
		if filter.InstanceIp != "" {
			db = db.Where("instance_ip like ?", "%"+escapeLike(filter.InstanceIp)+"%")
		}
		if filter.LastStatus != nil {
			db = db.Where("last_status = ?", *filter.LastStatus)
		}
		if filter.Enabled != "" {
			db = db.Where("enabled = ?", filter.Enabled)
		}
		return db.Order("id desc")
	})
	if err != nil {
		log.Println("查询采集目标失败：", err)
		return result, errors.New("查询采集目标失败")
	}
	return result, nil
}

// Add 新增采集目标
func (s *TargetService) Add(target *models.OpsMonitorTarget) error {
	instance, err := loadInstance(target.InstanceId)
	if err != nil {
		return err
	}
	if err := s.checkUnique(instance.ID, target.InstanceId); err != nil {
		return err
	}
	applyTargetDefaults(target)
	target.InstanceName = instance.Name
	target.InstanceIp = instance.Ip
	target.LastStatus = models.CollectStatusNone
	if err := models.DB.Save(target).Error; err != nil {
		log.Println("新增采集目标失败：", err)
		return errors.New("新增采集目标失败")
	}
	return nil
}

// Edit 编辑采集目标
func (s *TargetService) Edit(target *models.OpsMonitorTarget) error {
	var existing models.OpsMonitorTarget
	if err := models.DB.Where("id = ? AND del_flag = ?", target.ID, "0").First(&existing).Error; err != nil {
		return errors.New("采集目标不存在")
	}
	instance, err := loadInstance(target.InstanceId)
	if err != nil {
		return err
	}
	if err := s.checkUnique(existing.ID, target.InstanceId); err != nil {
		return err
	}
	applyTargetDefaults(target)
	target.InstanceName = instance.Name
	target.InstanceIp = instance.Ip
	target.CreateBy = existing.CreateBy
	target.DelFlag = existing.DelFlag
	if err := models.DB.Model(&existing).Updates(map[string]interface{}{
		"instance_id":   target.InstanceId,
		"instance_name": target.InstanceName,
		"instance_ip":   target.InstanceIp,
		"key_id":        target.KeyId,
		"enabled":       target.Enabled,
		"interval_sec":  target.IntervalSec,
		"timeout_sec":   target.TimeoutSec,
		"remark":        target.Remark,
		"update_by":     target.UpdateBy,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		log.Println("编辑采集目标失败：", err)
		return errors.New("编辑采集目标失败")
	}
	// 换主机后旧主机的序列需要清理，避免面板残留
	if existing.InstanceIp != "" && existing.InstanceIp != target.InstanceIp {
		deleteInstanceSeries(existing.InstanceIp)
	}
	return nil
}

// Delete 软删除采集目标并清理其在数据源中的序列
func (s *TargetService) Delete(id int) error {
	var existing models.OpsMonitorTarget
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&existing).Error; err != nil {
		return errors.New("采集目标不存在")
	}
	if err := models.DB.Model(&existing).Update("del_flag", "1").Error; err != nil {
		log.Println("删除采集目标失败：", err)
		return errors.New("删除采集目标失败")
	}
	deleteInstanceSeries(existing.InstanceIp)
	return nil
}

// Toggle 启用/停用采集目标
func (s *TargetService) Toggle(id int, enabled string) error {
	if enabled != "0" && enabled != "1" {
		return errors.New("启用状态非法")
	}
	if err := models.DB.Model(&models.OpsMonitorTarget{}).Where("id = ? AND del_flag = ?", id, "0").
		Update("enabled", enabled).Error; err != nil {
		log.Println("更新采集目标状态失败：", err)
		return errors.New("更新采集目标状态失败")
	}
	if enabled == "0" {
		var existing models.OpsMonitorTarget
		if err := models.DB.Where("id = ?", id).First(&existing).Error; err == nil {
			deleteInstanceSeries(existing.InstanceIp)
		}
	}
	return nil
}

// Sync 从主机列表补齐采集目标，返回新增数量
func (s *TargetService) Sync(createBy string) (int, error) {
	var instances []models.OpsInstance
	if err := models.DB.Where("del_flag = ? AND status = ?", "0", "1").Find(&instances).Error; err != nil {
		log.Println("查询主机列表失败：", err)
		return 0, errors.New("查询主机列表失败")
	}

	var existing []models.OpsMonitorTarget
	if err := models.DB.Where("del_flag = ?", "0").Find(&existing).Error; err != nil {
		log.Println("查询采集目标失败：", err)
		return 0, errors.New("同步采集目标失败")
	}
	covered := make(map[int]bool, len(existing))
	for _, item := range existing {
		covered[item.InstanceId] = true
	}

	created := 0
	for _, item := range instances {
		if covered[item.ID] {
			continue
		}
		target := &models.OpsMonitorTarget{
			InstanceId:   item.ID,
			InstanceName: item.Name,
			InstanceIp:   item.Ip,
			KeyId:        0,
			Enabled:      "1",
		}
		applyTargetDefaults(target)
		target.CreateBy = createBy
		target.UpdateBy = createBy
		target.LastStatus = models.CollectStatusNone
		if err := models.DB.Save(target).Error; err != nil {
			log.Printf("同步采集目标失败 instance=%d：%v", item.ID, err)
			continue
		}
		created++
	}
	return created, nil
}

// TargetKeyOption 采集目标表单的凭证选项，只返回非敏感字段
type TargetKeyOption struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	User string `json:"user"`
}

// InstanceKeys 查询指定主机已绑定的可用凭证，用于采集目标表单凭证下拉联动
func (s *TargetService) InstanceKeys(instanceId int) ([]TargetKeyOption, error) {
	if instanceId <= 0 {
		return nil, errors.New("缺少主机ID")
	}
	var keys []models.OpsKey
	err := models.DB.Model(&models.OpsKey{}).
		Select("ops_key.id, ops_key.name, ops_key.user").
		Joins("JOIN ops_instance_keys ON ops_instance_keys.key_id = ops_key.id").
		Where("ops_instance_keys.instance_id = ? AND ops_key.del_flag = ? AND ops_key.status = ?", instanceId, "0", "1").
		Order("ops_key.id").
		Find(&keys).Error
	if err != nil {
		log.Println("查询主机绑定凭证失败：", err)
		return nil, errors.New("查询主机绑定凭证失败")
	}
	options := make([]TargetKeyOption, 0, len(keys))
	for _, key := range keys {
		options = append(options, TargetKeyOption{Id: key.ID, Name: key.Name, User: key.User})
	}
	return options, nil
}

func (s *TargetService) checkUnique(id, instanceId int) error {
	var count int64
	err := models.DB.Model(&models.OpsMonitorTarget{}).
		Where("del_flag = ? AND instance_id = ? AND id != ?", "0", instanceId, id).Count(&count).Error
	if err != nil {
		log.Println("校验采集目标失败：", err)
		return errors.New("校验采集目标失败")
	}
	if count > 0 {
		return errors.New("该主机已配置采集目标")
	}
	return nil
}

func loadInstance(instanceId int) (*models.OpsInstance, error) {
	var instance models.OpsInstance
	if err := models.DB.Where("id = ? AND del_flag = ?", instanceId, "0").First(&instance).Error; err != nil {
		return nil, errors.New("主机不存在")
	}
	if strings.TrimSpace(instance.Ip) == "" {
		return nil, errors.New("主机缺少 IP 地址，无法采集")
	}
	return &instance, nil
}

func applyTargetDefaults(target *models.OpsMonitorTarget) {
	if target.IntervalSec <= 0 {
		target.IntervalSec = collectIntervalSec()
	}
	if target.TimeoutSec <= 0 {
		target.TimeoutSec = collectTimeoutSec()
	}
	if target.Enabled != "0" && target.Enabled != "1" {
		target.Enabled = "1"
	}
	if target.KeyId < 0 {
		target.KeyId = 0
	}
}

// deleteInstanceSeries 清理 Pushgateway 中该主机的指标分组，失败仅记录日志
func deleteInstanceSeries(instanceIp string) {
	if strings.TrimSpace(instanceIp) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pushTimeoutSec())*time.Second)
	defer cancel()
	if err := datasource.NewPushgateway().Delete(ctx, instanceIp); err != nil {
		log.Printf("清理监控指标分组失败 instance=%s：%v", instanceIp, err)
	}
}

// channelIdsText 规范化渠道ID列表，仅保留正整数
func channelIdsText(raw string) string {
	kept := make([]string, 0)
	for _, field := range strings.Split(raw, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		id, err := strconv.Atoi(field)
		if err != nil || id <= 0 {
			continue
		}
		kept = append(kept, strconv.Itoa(id))
	}
	return strings.Join(kept, ",")
}
