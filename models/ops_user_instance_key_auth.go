package models

// OpsUserInstanceKeyAuth 用户-主机-密钥授权关系
type OpsUserInstanceKeyAuth struct {
	UserId     int    `gorm:"column:user_id;type:int(11);not null;comment:用户ID" json:"userId"`
	InstanceId int    `gorm:"column:instance_id;type:int(11);not null;comment:主机ID" json:"instanceId"`
	GroupId    int    `gorm:"column:group_id;type:int(11);not null;comment:主机分组ID" json:"groupId"`
	KeyId      int    `gorm:"column:key_id;type:int(11);not null;comment:密钥ID" json:"keyId"`
	AuthType   int    `gorm:"column:auth_type;type:int(1);not null;comment:授权类型（1 主机 2 主机组）" json:"authType"`
	DelFlag    string `gorm:"column:del_flag;type:varchar(1);default:0;comment:删除标识（0正常 1 已删除）" json:"delFlag"`
}

const TableOpsUserInstanceKeyAuth = "ops_user_instance_key_auth"

func (*OpsUserInstanceKeyAuth) TableName() string {
	return TableOpsUserInstanceKeyAuth
}
