package models

// OpsUserInstanceAuth 用户-主机/分组授权关系
type OpsUserInstanceAuth struct {
	UserId     int    `gorm:"column:user_id;type:int(11);not null;comment:用户ID" json:"userId"`
	InstanceId int    `gorm:"column:instance_id;type:int(11);default null;comment:主机ID" json:"instanceId"`
	GroupId    int    `gorm:"column:group_id;type:int(11);default null;comment:分组ID" json:"groupId"`
	AuthType   int    `gorm:"column:auth_type;type:int(11);not null;comment:类型: 1 主机 2 分组" json:"authType"`
	DelFlag    string `gorm:"column:del_flag;type:varchar(1);default:0;comment:删除标识（0正常 1 已删除）" json:"delFlag"`
}

const TableOpsUserInstanceAuth = "ops_user_instance_auth"

func (*OpsUserInstanceAuth) TableName() string {
	return TableOpsUserInstanceAuth
}
