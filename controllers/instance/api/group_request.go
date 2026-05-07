package api

import (
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/models"
)

type AddGroupRequest struct {
	Name     string `json:"name"`
	ParentId string `json:"parentId"`
	CreateBy string `json:"createBy"`
	UpdateBy string `json:"updateBy"`
	Remark   string `json:"remark"`
}

type UpdateGroupRequest struct {
	Id int `json:"id"`
	AddGroupRequest
}

type GroupInstanceRequest struct {
	GroupId     int    `json:"groupId"`
	InstanceIds []int  `json:"instanceIds"`
	OpsType     string `json:"opsType"` // add or remove 操作类型
}

type PageGroupInstanceRequest struct {
	GroupId int `json:"groupId"`
	controllers.PageRequest
}

type PageGroupInstanceResponse struct {
	Total     int64                `json:"total"`
	PageNum   int                  `json:"pageNum"`
	PageSize  int                  `json:"pageSize"`
	TotalPage int                  `json:"totalPage"`
	Data      []models.OpsInstance `json:"data"`
}

// 扫描主机请求
type ScanHostsRequest struct {
	IpRange string `json:"ipRange" binding:"required"` // IP网段，如 192.168.1.0/24 或 192.168.1.1-100
}

// 扫描到的主机信息
type ScannedHost struct {
	Ip     string `json:"ip"`
	Port   int    `json:"port"`
	OsType string `json:"osType"` // Linux 或 Windows
}

// 扫描主机响应
type ScanHostsResponse struct {
	Hosts []ScannedHost `json:"hosts"`
}

// 保存扫描主机请求
type SaveScannedHostsRequest struct {
	GroupId int            `json:"groupId" binding:"required"`
	Hosts   []ScannedHost  `json:"hosts" binding:"required"`
}
