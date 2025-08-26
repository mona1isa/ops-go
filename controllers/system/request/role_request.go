package request

import "github.com/zhany/ops-go/controllers"

type RoleRequest struct {
	Name     string `json:"name" binding:"required"`
	OrderNum int    `json:"orderNum"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
}

type EditRoleRequest struct {
	Id       int    `json:"id" binding:"required"`
	Name     string `json:"name"`
	OrderNum int    `json:"orderNum"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
}

type PageRoleRequest struct {
	controllers.PageRequest
	Name     string `json:"name"`
	OrderNum int    `json:"orderNum"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
}
