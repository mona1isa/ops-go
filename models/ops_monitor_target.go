package models

import "time"

// 采集状态
const (
	CollectStatusNone = 0 // 未采集
	CollectStatusOk   = 1 // 采集成功
	CollectStatusFail = 2 // 采集失败
)

// OpsMonitorTarget 监控采集目标
type OpsMonitorTarget struct {
	InstanceId    int        `gorm:"column:instance_id;type:int(11);not null;index:idx_monitor_target_instance;comment:主机ID" json:"instanceId"`
	InstanceName  string     `gorm:"column:instance_name;type:varchar(32);not null;comment:主机名称" json:"instanceName"`
	InstanceIp    string     `gorm:"column:instance_ip;type:varchar(32);not null;comment:IP地址" json:"instanceIp"`
	KeyId         int        `gorm:"column:key_id;type:int(11);default:0;comment:采集凭证ID（0 自动选择）" json:"keyId"`
	Enabled       string     `gorm:"column:enabled;type:varchar(1);default:1;comment:采集开关（1 启用 0 停用）" json:"enabled"`
	IntervalSec   int        `gorm:"column:interval_sec;type:int(11);default:60;comment:采集间隔(秒)" json:"intervalSec"`
	TimeoutSec    int        `gorm:"column:timeout_sec;type:int(11);default:15;comment:采集超时(秒)" json:"timeoutSec"`
	LastCollectAt *time.Time `gorm:"column:last_collect_at;comment:最近采集成功时间" json:"lastCollectAt"`
	LastStatus    int8       `gorm:"column:last_status;type:tinyint;default:0;comment:采集状态（0 未采集 1 成功 2 失败）" json:"lastStatus"`
	LastError     string     `gorm:"column:last_error;type:varchar(500);comment:最近采集失败原因" json:"lastError"`
	Base
}

const TableOpsMonitorTarget = "ops_monitor_target"

func (OpsMonitorTarget) TableName() string {
	return TableOpsMonitorTarget
}
