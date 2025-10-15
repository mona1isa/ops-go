package models

// OpsUserInstanceAuth 用户-主机/分组授权关系
type OpsUserInstanceAuth struct {
	UserId     int    `gorm:"type:int(11);not null;comment:用户ID" json:"user_id"`
	InstanceId int    `gorm:"type:int(11);default null;comment:主机ID" json:"instance_id"`
	GroupId    int    `gorm:"type:int(11);default null;comment:分组ID" json:"group_id"`
	AuthType   int    `gorm:"type:int(11);not null;comment:类型: 1 主机 2 分组" json:"auth_type"`
	DelFlag    string `gorm:"type:varchar(1);default:0;comment:删除标识（0正常 1 已删除）" json:"delFlag"`
}

const TableOpsUserInstanceAuth = "ops_user_instance_auth"

func (*OpsUserInstanceAuth) TableName() string {
	return TableOpsUserInstanceAuth
}
