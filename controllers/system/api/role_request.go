package api

import "github.com/zhany/ops-go/controllers"

type RoleRequest struct {
	Name     string `json:"name" binding:"required"`
	OrderNum int    `json:"orderNum"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
	MenuIds  []int  `json:"menuIds" binding:"required"` // 菜单ID
}

type EditRoleRequest struct {
	Id       int    `json:"id" binding:"required"`
	Name     string `json:"name"`
	OrderNum int    `json:"orderNum"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
	MenuIds  []int  `json:"menuIds"` // 菜单ID
}

type PageRoleRequest struct {
	controllers.PageRequest
	Name     string `json:"name"`
	OrderNum int    `json:"orderNum"`
	Status   string `json:"status"`
	Remark   string `json:"remark"`
}

type RoleAsignRequest struct {
	UserIds []int `json:"userIds" binding:"required"`
	RoleId  int   `json:"roleId" binding:"required"`
}
