package api

import "github.com/zhany/ops-go/controllers"

type AddInstanceRequest struct {
	Name        string `json:"name"`
	Cpu         int    `json:"cpu"`
	MemMb       int    `json:"memMb"`
	DiskGb      int    `json:"diskGb"`
	Ip          string `json:"ip"`
	Os          string `json:"os"`
	Status      string `json:"status"`
	CreateBy    string `json:"createBy"`
	UpdateBy    string `json:"updateBy"`
	DelFlag     string `json:"delFlag"`
	Remark      string `json:"remark"`
	BindingKeys []int  `json:"bindingKeys"`
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

type InstanceKeyBindingRequest struct {
	InstanceId int `json:"instanceId" required:"true"`
	KeyId      int `json:"keyId" required:"true"`
}

type InstanceKeyUnbindingRequest struct {
	InstanceId int   `json:"instanceId" required:"true"`
	KeyIds     []int `json:"keyIds" required:"true"`
}

type ListInstanceRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type OsTypeRequest struct {
	OsType string `json:"osType"`
}
