package models

// OpsInstanceKey 主机-密钥关联
type OpsInstanceKey struct {
	InstanceId int `gorm:"column:instance_id;type:int(11);not null;comment:主机ID;index:idx_instance_id_key_id" json:"instance_id"` // ops_instance表主键
	KeyId      int `gorm:"column:key_id;type:int(11);not null;comment:密钥ID;index:idx_instance_id_key_id" json:"key_id"`           // ops_key表主键
}
