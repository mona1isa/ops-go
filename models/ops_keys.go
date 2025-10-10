package models

import (
	"github.com/zhany/ops-go/utils"
)

// OpsKey 主机密钥
type OpsKey struct {
	Name        string `gorm:"column:name;type:varchar(128);not null;uniqueIndex:uk_name;comment:名称" json:"name"`
	User        string `gorm:"column:user;type:varchar(128);not null;comment:登录用户" json:"user"`
	Credentials string `gorm:"column:credentials;type:text;not null;comment:登录凭证" json:"credentials"`
	Status      string `gorm:"column:status;type:varchar(1);default:1;comment:状态（1 正常 0 禁用）" json:"status"`
	Protocol    string `gorm:"column:protocol;type:varchar(16);not null;comment:协议: SSH or RDP" json:"protocol"`
	Port        int    `gorm:"column:port;type:int(11);not null;comment:端口" json:"port"`
	Type        int    `gorm:"column:type;type:int(11);not null;comment:类型: 1 密码 2 秘钥" json:"type"`
	Base
}

const TableOpsKey = "ops_key"

func (OpsKey) TableName() string {
	return TableOpsKey
}

// GetPlainCredentials 获取明文凭证
func (k *OpsKey) GetPlainCredentials() (credentials string, err error) {
	key, err := utils.DecryptKey(k.Credentials)
	if err != nil {
		return "", err
	}
	return key, nil
}
