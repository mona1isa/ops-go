package models

import "time"

// OpsCommandBlockLog 高危指令拦截记录（只追加，不做逻辑删除）
// 此前拦截命中只写进程日志，事后无法追溯是谁在哪台主机上被拦过什么命令。
type OpsCommandBlockLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID    string    `gorm:"type:varchar(64);index;comment:会话ID" json:"sessionId"`
	Channel      int8      `gorm:"type:smallint;default:1;comment:接入渠道 1 SSH堡垒机 2 Web终端" json:"channel"`
	UserID       int       `gorm:"index;comment:平台用户ID" json:"userId"`
	UserName     string    `gorm:"type:varchar(32);comment:平台登录名" json:"userName"`
	InstanceID   int       `gorm:"index;comment:主机ID" json:"instanceId"`
	InstanceName string    `gorm:"type:varchar(32);comment:主机名称" json:"instanceName"`
	InstanceIP   string    `gorm:"type:varchar(32);comment:主机IP" json:"instanceIp"`
	KeyUser      string    `gorm:"type:varchar(32);comment:远端登录账号" json:"keyUser"`
	Command      string    `gorm:"type:varchar(1000);comment:被拦截的命令原文" json:"command"`
	Normalized   string    `gorm:"type:varchar(1000);comment:归一化后命中规则的形态" json:"normalized"`
	RuleID       int       `gorm:"index;comment:命中的规则ID" json:"ruleId"`
	RuleName     string    `gorm:"type:varchar(64);comment:命中的规则名称" json:"ruleName"`
	CreatedAt    time.Time `gorm:"index;not null;comment:拦截时间" json:"createdAt"`
}

const (
	TableOpsCommandBlockLog = "ops_command_block_log"

	BlockChannelBastion int8 = 1 // SSH 堡垒机 :2222
	BlockChannelWebTerm int8 = 2 // Web 终端
)

func (OpsCommandBlockLog) TableName() string {
	return TableOpsCommandBlockLog
}
