package models

// OpsInstanceGroup 主机-分组关联表
type OpsInstanceGroup struct {
	InstanceId int `gorm:"type:varchar(11);not null;comment:实例ID" json:"instanceId"` // ops_instance表主键
	GroupId    int `gorm:"type:varchar(11);not null;comment:分组ID" json:"groupId"`    // ops_group表主键
}

const TableOpsInstanceGroup = "ops_instance_group"

func (OpsInstanceGroup) TableName() string {
	return TableOpsInstanceGroup
}
