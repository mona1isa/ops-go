package models

import (
	"fmt"
	"github.com/zhany/ops-go/utils"
	"gorm.io/gorm"
)

type OpsInstance struct {
	DeptId     int    `gorm:"type:int(11);not null;comment:部门ID" json:"deptId"`
	InstanceId string `gorm:"type:varchar(32);not null;comment:实例ID" json:"instanceId"`
	Name       string `gorm:"type:varchar(32);not null;unique;comment:主机名称" json:"name"`
	Cpu        int    `gorm:"type:int(11);not null;comment:CPU核数" json:"cpu"`
	Mem        int    `gorm:"column:mem_mb;type:int(11);not null;comment:内存大小(MB)" json:"memMb"`
	Disk       int    `gorm:"column:disk_gb;type:int(11);not null;comment:磁盘大小(GB)" json:"diskGb"`
	Ip         string `gorm:"type:varchar(32);not null;comment:IP地址" json:"ip"`
	Port       int    `gorm:"type:int(11);not null;comment:端口" json:"port"`
	Os         string `gorm:"type:varchar(32);not null;comment:操作系统" json:"os"`
	Status     string `gorm:"type:varchar(1);default:1;comment:状态（1 正常 0 禁用）" json:"status"`
	Base
	DeptName string `gorm:"-" json:"deptName"` //部门名称
	Spec     string `gorm:"-" json:"spec"`     //规格
}

const TableOpsInstance = "ops_instance"

func (OpsInstance) TableName() string {
	return TableOpsInstance
}

func (i *OpsInstance) BeforeCreate(db *gorm.DB) (err error) {
	uuid := utils.GenUuid()
	i.InstanceId = uuid
	return
}

func (i *OpsInstance) AfterFind(db *gorm.DB) (err error) {
	DB.Model(&SysDept{}).Where("id = ?", i.DeptId).Select("name").Find(&i.DeptName)
	i.Spec = i.GetSpec()
	return
}

// GetSpec 格式化主机规格
func (i *OpsInstance) GetSpec() string {
	// Format mem
	var memStr string
	if i.Mem < 1024 {
		memStr = fmt.Sprintf("%dM", i.Mem)
	} else if i.Mem < 1024*1024 {
		memStr = fmt.Sprintf("%dG", i.Mem/1024)
	} else {
		memStr = fmt.Sprintf("%dT", i.Mem/(1024*1024))
	}
	// Format disk
	var diskStr string
	if i.Disk < 1024 {
		diskStr = fmt.Sprintf("%dG", i.Disk)
	} else if i.Disk < 1024*1024 {
		diskStr = fmt.Sprintf("%dT", i.Disk/1024)
	} else {
		diskStr = fmt.Sprintf("%dP", i.Disk/(1024*1024))
	}

	return fmt.Sprintf("%dC%s%s", i.Cpu, memStr, diskStr)
}
