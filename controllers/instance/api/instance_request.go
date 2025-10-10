package api

import "github.com/zhany/ops-go/controllers"

type AddInstanceRequest struct {
	Name     string `json:"name"`
	Cpu      int    `json:"cpu"`
	MemMb    int    `json:"memMb"`
	DiskGb   int    `json:"diskGb"`
	Ip       string `json:"ip"`
	Port     int    `json:"port"`
	Os       string `json:"os"`
	Status   string `json:"status"`
	CreateBy string `json:"createBy"`
	UpdateBy string `json:"updateBy"`
	DelFlag  string `json:"delFlag"`
	Remark   string `json:"remark"`
}

type UpdateInstanceRequest struct {
	Id int `json:"id" required:"true"`
	AddInstanceRequest
}

type ChangeStatusRequest struct {
	Id     int    `json:"id" required:"true"`
	Status string `json:"status" required:"true"`
}

type PageInstanceRequest struct {
	controllers.PageRequest
	AddInstanceRequest
}
