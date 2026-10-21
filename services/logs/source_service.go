package logs

import (
	"encoding/json"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/zhany/ops-go/controllers/logs/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services/logs/datasource"
	"gorm.io/gorm"
)

// LogSourceService 日志采集源管理
type LogSourceService struct{}

// List 采集源分页
func (s *LogSourceService) List(pageNum, pageSize int, filter api.LogSourcePageRequest) (models.PageResult[models.OpsLogSource], error) {
	result, err := models.Paginate[models.OpsLogSource](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		db = db.Where("del_flag = ?", "0")
		if filter.Name != "" {
			db = db.Where("name like ?", "%"+escapeLike(filter.Name)+"%")
		}
		if filter.InstanceIp != "" {
			db = db.Where("instance_ip like ?", "%"+escapeLike(filter.InstanceIp)+"%")
		}
		if filter.Path != "" {
			db = db.Where("path like ?", "%"+escapeLike(filter.Path)+"%")
		}
		if filter.Job != "" {
			db = db.Where("job = ?", filter.Job)
		}
		if filter.Enabled != "" {
			db = db.Where("enabled = ?", filter.Enabled)
		}
		if filter.LastStatus != nil {
			db = db.Where("last_status = ?", *filter.LastStatus)
		}
		return db.Order("id desc")
	})
	if err != nil {
		log.Println("查询日志采集源失败：", err)
		return result, errors.New("查询日志采集源失败")
	}
	return result, nil
}

// Get 采集源详情
func (s *LogSourceService) Get(id int) (*models.OpsLogSource, error) {
	var source models.OpsLogSource
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&source).Error; err != nil {
		return nil, errors.New("采集源不存在")
	}
	return &source, nil
}

// Add 新增采集源
func (s *LogSourceService) Add(source *models.OpsLogSource) error {
	if err := normalizeSource(source); err != nil {
		return err
	}
	if err := s.checkUnique(0, source.InstanceId, source.Path); err != nil {
		return err
	}
	if err := models.DB.Save(source).Error; err != nil {
		log.Println("新增日志采集源失败：", err)
		return errors.New("新增采集源失败")
	}
	return nil
}

// Edit 更新采集源。主机/路径/凭证变化会重置读取位点，避免沿用旧文件的偏移量。
func (s *LogSourceService) Edit(request api.LogSourceRequest) error {
	var existing models.OpsLogSource
	if err := models.DB.Where("id = ? AND del_flag = ?", request.Id, "0").First(&existing).Error; err != nil {
		return errors.New("采集源不存在")
	}

	merged := existing
	merged.Name = request.Name
	merged.InstanceId = request.InstanceId
	merged.KeyId = request.KeyId
	merged.Path = request.Path
	merged.Job = request.Job
	merged.Source = request.Source
	merged.Labels = request.Labels
	merged.LevelRegex = request.LevelRegex
	merged.Enabled = request.Enabled
	merged.IntervalSec = request.IntervalSec
	merged.TimeoutSec = request.TimeoutSec
	merged.MaxBytes = request.MaxBytes
	merged.BackBytes = request.BackBytes
	merged.Remark = request.Remark
	// 创建人不可被编辑覆盖
	merged.CreateBy = existing.CreateBy

	if err := normalizeSource(&merged); err != nil {
		return err
	}

	updates := map[string]any{
		"name": merged.Name, "instance_id": merged.InstanceId, "instance_name": merged.InstanceName,
		"instance_ip": merged.InstanceIp, "key_id": merged.KeyId, "path": merged.Path,
		"job": merged.Job, "source": merged.Source, "labels": merged.Labels, "level_regex": merged.LevelRegex,
		"enabled": merged.Enabled, "interval_sec": merged.IntervalSec, "timeout_sec": merged.TimeoutSec,
		"max_bytes": merged.MaxBytes, "back_bytes": merged.BackBytes, "remark": merged.Remark,
		"create_by": merged.CreateBy, "updated_at": time.Now(),
	}
	if merged.InstanceId != existing.InstanceId || merged.Path != existing.Path || merged.KeyId != existing.KeyId {
		updates["last_inode"] = int64(0)
		updates["last_pos"] = int64(0)
	}

	if err := models.DB.Model(&models.OpsLogSource{}).Where("id = ?", request.Id).Updates(updates).Error; err != nil {
		log.Println("更新日志采集源失败：", err)
		return errors.New("更新采集源失败")
	}
	return nil
}

// Delete 软删除采集源
func (s *LogSourceService) Delete(id int) error {
	if err := models.DB.Model(&models.OpsLogSource{}).Where("id = ? AND del_flag = ?", id, "0").
		Update("del_flag", "1").Error; err != nil {
		log.Println("删除日志采集源失败：", err)
		return errors.New("删除采集源失败")
	}
	return nil
}

// Toggle 启停采集源
func (s *LogSourceService) Toggle(id int, enabled string) error {
	if enabled != "0" && enabled != "1" {
		return errors.New("启停状态非法")
	}
	return models.DB.Model(&models.OpsLogSource{}).Where("id = ? AND del_flag = ?", id, "0").
		Updates(map[string]any{"enabled": enabled, "updated_at": time.Now()}).Error
}

// ResetOffset 重新定位读取位点：清零后下一轮按「首次接入」策略回溯尾部内容
func (s *LogSourceService) ResetOffset(id int) error {
	return models.DB.Model(&models.OpsLogSource{}).Where("id = ?", id).
		Updates(map[string]any{"last_inode": int64(0), "last_pos": int64(0)}).Error
}

// Jobs 返回已配置的 job 标签集合，供检索页下拉
func (s *LogSourceService) Jobs() ([]string, error) {
	var jobs []string
	if err := models.DB.Model(&models.OpsLogSource{}).
		Where("del_flag = ?", "0").Distinct().Pluck("job", &jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// SourceKeyOption 采集凭证下拉项
type SourceKeyOption struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	User string `json:"user"`
}

// InstanceKeys 查询指定主机绑定的凭证，供采集源表单凭证下拉联动
func (s *LogSourceService) InstanceKeys(instanceId int) ([]SourceKeyOption, error) {
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
	options := make([]SourceKeyOption, 0, len(keys))
	for _, key := range keys {
		options = append(options, SourceKeyOption{Id: key.ID, Name: key.Name, User: key.User})
	}
	return options, nil
}

// Option 采集源下拉项，检索页按采集源快速过滤
type Option struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	InstanceIp string `json:"instanceIp"`
	Path       string `json:"path"`
	Job        string `json:"job"`
	Source     string `json:"source"`
}

// Options 启用中的采集源精简列表
func (s *LogSourceService) Options() ([]Option, error) {
	var sources []models.OpsLogSource
	if err := models.DB.Where("del_flag = ? AND enabled = ?", "0", "1").Order("id asc").Find(&sources).Error; err != nil {
		return nil, errors.New("查询采集源失败")
	}
	options := make([]Option, 0, len(sources))
	for _, source := range sources {
		options = append(options, Option{
			Id: source.ID, Name: source.Name, InstanceIp: source.InstanceIp,
			Path: source.Path, Job: source.Job, Source: source.Source,
		})
	}
	return options, nil
}

// checkUnique 同一主机同一路径只允许一个采集源，避免重复写入相同日志
func (s *LogSourceService) checkUnique(id, instanceId int, path string) error {
	var count int64
	err := models.DB.Model(&models.OpsLogSource{}).
		Where("del_flag = ? AND instance_id = ? AND path = ? AND id != ?", "0", instanceId, path, id).
		Count(&count).Error
	if err != nil {
		log.Println("校验日志采集源唯一性失败：", err)
		return errors.New("校验采集源失败")
	}
	if count > 0 {
		return errors.New("该主机已存在相同路径的采集源")
	}
	return nil
}

// normalizeSource 校验并补齐采集源默认值
func normalizeSource(source *models.OpsLogSource) error {
	source.Name = strings.TrimSpace(source.Name)
	source.Path = strings.TrimSpace(source.Path)
	if source.Name == "" {
		return errors.New("采集任务名称不能为空")
	}
	if source.InstanceId <= 0 {
		return errors.New("请选择采集主机")
	}
	if source.Path == "" {
		return errors.New("日志文件路径不能为空")
	}
	if !strings.HasPrefix(source.Path, "/") {
		return errors.New("日志文件路径必须为绝对路径")
	}
	if len(source.Path) > 512 {
		return errors.New("日志文件路径过长")
	}

	var host models.OpsInstance
	if err := models.DB.Where("id = ? AND del_flag = ?", source.InstanceId, "0").First(&host).Error; err != nil {
		return errors.New("采集主机不存在")
	}
	source.InstanceName = host.Name
	source.InstanceIp = host.Ip

	if source.Job == "" {
		source.Job = "ops_log"
	}
	source.Job = datasource.SafeLabelValue(source.Job)
	if source.Job == "" {
		return errors.New("job 标签为空或含非法字符")
	}
	source.Source = datasource.SafeLabelValue(source.Source)
	if err := validateLevelRegex(source.LevelRegex); err != nil {
		return err
	}
	source.Labels = normalizeLabels(source.Labels)

	if source.IntervalSec <= 0 {
		source.IntervalSec = datasource.CollectIntervalSec()
	}
	if source.TimeoutSec <= 0 {
		source.TimeoutSec = datasource.CollectTimeoutSec()
	}
	if source.Enabled == "0" {
		source.Enabled = "0"
	} else {
		source.Enabled = "1"
	}
	return nil
}

func validateLevelRegex(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}
	if len(pattern) > 255 {
		return errors.New("级别正则过长")
	}
	if strings.Contains(pattern, "\n") {
		return errors.New("级别正则不能包含换行")
	}
	if !hasCaptureGroup(pattern) {
		return errors.New("级别正则需包含一个捕获组，如 (ERROR|WARN)")
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return errors.New("级别正则无法编译")
	}
	return nil
}

// hasCaptureGroup 粗校验：存在未被转义的左括号即认为有捕获组，具体合法性交由 Go 正则引擎
func hasCaptureGroup(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '\\' {
			i++
			continue
		}
		if pattern[i] == '(' {
			return true
		}
	}
	return false
}

// normalizeLabels 校验并规范化自定义标签 JSON，非法项丢弃而不是整体报错
func normalizeLabels(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	labels := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return ""
	}
	clean := make(map[string]string, len(labels))
	for key, value := range labels {
		if !validLabelName(key) || reservedLabel[key] {
			continue
		}
		val := datasource.SafeLabelValue(value)
		if val == "" {
			continue
		}
		clean[key] = val
	}
	if len(clean) == 0 {
		return ""
	}
	out, err := json.Marshal(clean)
	if err != nil {
		return ""
	}
	return string(out)
}

// validLabelName Loki 标签名只允许字母、数字、下划线，且不能以数字开头
func validLabelName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 32 {
		return false
	}
	if first := name[0]; !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		default:
			return false
		}
	}
	return true
}

// escapeLike 转义 LIKE 通配符，避免用户输入的 % 和 _ 影响匹配范围
func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
