package models

// OpsUserInstanceKeyAuth 用户-主机-密钥授权关系
type OpsUserInstanceKeyAuth struct {
	UserId     int    `gorm:"type:int(11);not null;comment:用户ID" json:"user_id"`
	InstanceId int    `gorm:"type:int(11);not null;comment:主机ID" json:"instance_id"`
	KeyId      int    `gorm:"type:int(11);not null;comment:密钥ID" json:"key_id"`
	DelFlag    string `gorm:"type:varchar(1);default:0;comment:删除标识（0正常 1 已删除）" json:"delFlag"`
}

const TableOpsUserInstanceKeyAuth = "ops_user_instance_key_auth"

func (*OpsUserInstanceKeyAuth) TableName() string {
	return TableOpsUserInstanceKeyAuth
}
