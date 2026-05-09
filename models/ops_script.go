package models

type OpsScript struct {
	Name    string `gorm:"type:varchar(128);not null;comment:脚本名称" json:"name"`
	Content string `gorm:"type:text;comment:脚本内容" json:"content"`
	Type    string `gorm:"type:varchar(20);default:'bash';comment:脚本类型" json:"type"`
	Remark  string `gorm:"type:varchar(500);comment:备注" json:"remark"`
	Base
}

const TableOpsScript = "ops_script"

func (OpsScript) TableName() string {
	return TableOpsScript
}
