package api

import "github.com/zhany/ops-go/controllers"

type AddKeysRequest struct {
	Name        string `json:"name" binding:"required"`
	User        string `json:"user" binding:"required"`
	Credentials string `json:"credentials" binding:"required"`
	Status      string `json:"status"`
	Protocol    string `json:"protocol" binding:"required"`
	Port        int    `json:"port"`
	Type        int    `json:"type"`
	Remark      string `json:"remark"`
	CreateBy    string `json:"createBy"`
	UpdateBy    string `json:"updateBy"`
}

type UpdateKeysRequest struct {
	Id int `json:"id" binding:"required"`
	AddKeysRequest
}

type PageKeysRequest struct {
	controllers.PageRequest
	Name     string `json:"name"`
	User     string `json:"user"`
	Status   string `json:"status"`
	Protocol string `json:"protocol"`
	Type     int    `json:"type"`
}
