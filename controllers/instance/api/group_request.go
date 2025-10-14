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
