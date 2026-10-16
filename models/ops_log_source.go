package models

import "time"

// OpsLogSource 日志采集源：一条「哪台主机的哪个文件」的增量拉取任务
type OpsLogSource struct {
	Name          string     `gorm:"column:name;type:varchar(128);not null;comment:采集任务名称" json:"name"`
	InstanceId    int        `gorm:"column:instance_id;type:int(11);not null;index:idx_log_source_instance;comment:主机ID" json:"instanceId"`
	InstanceName  string     `gorm:"column:instance_name;type:varchar(32);not null;comment:主机名称" json:"instanceName"`
	InstanceIp    string     `gorm:"column:instance_ip;type:varchar(32);not null;comment:主机IP" json:"instanceIp"`
	KeyId         int        `gorm:"column:key_id;type:int(11);default:0;comment:采集凭证ID（0 自动选择）" json:"keyId"`
	Path          string     `gorm:"column:path;type:varchar(512);not null;comment:日志文件路径，支持 * ? 通配符" json:"path"`
	Job           string     `gorm:"column:job;type:varchar(64);default:ops_log;comment:Loki job 标签" json:"job"`
	Source        string     `gorm:"column:source;type:varchar(64);comment:业务来源标签，如 nginx/access" json:"source"`
	Labels        string     `gorm:"column:labels;type:varchar(512);comment:附加标签(JSON 对象)" json:"labels"`
	LevelRegex    string     `gorm:"column:level_regex;type:varchar(255);comment:级别提取正则（需包含捕获组），为空时用内置 ERROR/WARN/INFO 识别" json:"levelRegex"`
	Enabled       string     `gorm:"column:enabled;type:varchar(1);default:1;comment:采集开关（1 启用 0 停用）" json:"enabled"`
	IntervalSec   int        `gorm:"column:interval_sec;type:int(11);default:15;comment:采集间隔(秒)" json:"intervalSec"`
	TimeoutSec    int        `gorm:"column:timeout_sec;type:int(11);default:20;comment:单次拉取超时(秒)" json:"timeoutSec"`
	MaxBytes      int        `gorm:"column:max_bytes;type:int(11);default:0;comment:单轮最大拉取字节(0 用全局默认)" json:"maxBytes"`
	BackBytes     int        `gorm:"column:back_bytes;type:int(11);default:0;comment:首次接入回溯字节(0 用全局默认)" json:"backBytes"`
	LastInode     int64      `gorm:"column:last_inode;type:bigint;default:0;comment:上次读取的文件 inode，用于识别轮转" json:"lastInode"`
	LastPos       int64      `gorm:"column:last_pos;type:bigint;default:0;comment:下次读取的起始字节偏移" json:"lastPos"`
	LastFile      string     `gorm:"column:last_file;type:varchar(512);comment:上次实际读取的文件" json:"lastFile"`
	LastLines     int        `gorm:"column:last_lines;type:int(11);default:0;comment:上次采集行数" json:"lastLines"`
	LastDrop      int64      `gorm:"column:last_drop;type:bigint;default:0;comment:上次因限速丢弃的字节数" json:"lastDrop"`
	LastCollectAt *time.Time `gorm:"column:last_collect_at;comment:最近采集成功时间" json:"lastCollectAt"`
	LastStatus    int8       `gorm:"column:last_status;type:tinyint;default:0;comment:采集状态（0 未采集 1 成功 2 失败）" json:"lastStatus"`
	LastError     string     `gorm:"column:last_error;type:varchar(500);comment:最近采集失败原因" json:"lastError"`
	Base
}

const TableOpsLogSource = "ops_log_source"

func (OpsLogSource) TableName() string {
	return TableOpsLogSource
}
