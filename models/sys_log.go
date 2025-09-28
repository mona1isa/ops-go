package models

type SysLog struct {
	DeptId     int    `gorm:"comment:创建人部门" json:"deptId"`
	CreateUser string `gorm:"type:varchar(16);comment:创建人用户名" json:"createUser"`
	Method     string `gorm:"type:varchar(8);comment:请求方法" json:"method"`
	RequestUri string `gorm:"type:varchar(255);comment:请求地址" json:"requestUri"`
	Params     string `gorm:"type:text;comment:请求参数" json:"params"`
	Resp       string `gorm:"type:text;comment:resp" json:"resp"`
	IpAddr     string `gorm:"type:varchar(20);comment:请求IP" json:"ipAddr"`
	StatusCode string `gorm:"type:char(8);comment:响应状态码" json:"statusCode"`
	CostTimeMs int64  `gorm:"comment:响应耗时" json:"costTimeMs"`
	Base
}

const TableSysLog = "sys_log"

func (SysLog) TableName() string {
	return TableSysLog
}
