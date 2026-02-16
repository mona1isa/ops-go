package models

import (
	"time"
)

// OpsSessionRecord 会话记录
type OpsSessionRecord struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID     string    `gorm:"type:varchar(64);uniqueIndex;not null;comment:会话ID" json:"sessionId"`
	UserID        int       `gorm:"index;not null;comment:用户ID" json:"userId"`
	InstanceID    int       `gorm:"index;not null;comment:主机ID" json:"instanceId"`
	InstanceName  string    `gorm:"type:varchar(32);not null;comment:主机名称" json:"instanceName"`
	InstanceIP    string    `gorm:"type:varchar(32);not null;comment:主机IP" json:"instanceIp"`
	KeyID         int       `gorm:"not null;comment:凭证ID" json:"keyId"`
	KeyName       string    `gorm:"type:varchar(32);not null;comment:凭证名称" json:"keyName"`
	KeyUser       string    `gorm:"type:varchar(32);not null;comment:登录用户" json:"keyUser"`
	StartTime     time.Time `gorm:"index;not null;comment:开始时间" json:"startTime"`
	EndTime       *time.Time `gorm:"comment:结束时间" json:"endTime"`
	Duration      int       `gorm:"default:0;comment:会话时长（秒）" json:"duration"`
	Status        int8      `gorm:"default:1;comment:状态（1 进行中 2 已结束 3 异常中断）" json:"status"`
	RecordingFile string    `gorm:"type:varchar(255);comment:录像文件路径" json:"recordingFile"`
	FileSize      int64     `gorm:"default:0;comment:文件大小（字节）" json:"fileSize"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

const (
	SessionStatusActive    int8 = 1 // 进行中
	SessionStatusCompleted int8 = 2 // 已结束
	SessionStatusAborted   int8 = 3 // 异常中断
)

const TableOpsSessionRecord = "ops_session_record"

func (OpsSessionRecord) TableName() string {
	return TableOpsSessionRecord
}
