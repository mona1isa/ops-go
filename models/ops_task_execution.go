package models

import "time"

// 执行类型
const (
	ExecTypeQuickCommand = 1 // 快速命令
	ExecTypeQuickScript  = 2 // 快速脚本
	ExecTypeQuickFile    = 3 // 快速文件
	ExecTypeTemplate     = 4 // 模板执行
	ExecTypePipeline     = 5 // 编排执行
)

// 执行状态
const (
	ExecStatusPending      = 1 // 待执行
	ExecStatusRunning      = 2 // 执行中
	ExecStatusCompleted    = 3 // 已完成
	ExecStatusPartialFail  = 4 // 部分失败
	ExecStatusAllFail      = 5 // 全部失败
	ExecStatusCancelled    = 6 // 已取消
)

// OpsTaskExecution 任务执行记录
type OpsTaskExecution struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ExecutionNo  string     `gorm:"type:varchar(64);uniqueIndex;not null;comment:执行编号" json:"executionNo"`
	Name         string     `gorm:"type:varchar(128);not null;comment:执行名称" json:"name"`
	Type         int8       `gorm:"type:tinyint;not null;comment:类型 1快速命令 2快速脚本 3快速文件 4模板执行 5编排执行" json:"type"`
	SourceId     int        `gorm:"type:int;comment:来源ID 模板ID/编排ID" json:"sourceId"`
	UserId       int        `gorm:"index;not null;comment:执行人ID" json:"userId"`
	UserName     string     `gorm:"type:varchar(32);not null;comment:执行人用户名" json:"userName"`
	Status       int8       `gorm:"type:tinyint;default:1;comment:状态 1待执行 2执行中 3已完成 4部分失败 5全部失败 6已取消" json:"status"`
	TotalHosts   int        `gorm:"type:int;default:0;comment:目标主机数" json:"totalHosts"`
	SuccessHosts int        `gorm:"type:int;default:0;comment:成功主机数" json:"successHosts"`
	FailHosts    int        `gorm:"type:int;default:0;comment:失败主机数" json:"failHosts"`
	Timeout      int        `gorm:"type:int;default:300;comment:超时时间(秒)" json:"timeout"`
	StartedAt    *time.Time `gorm:"comment:开始时间" json:"startedAt"`
	FinishedAt   *time.Time `gorm:"comment:结束时间" json:"finishedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}
