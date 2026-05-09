package api

import "github.com/zhany/ops-go/controllers"

type AddScriptRequest struct {
	Name    string `json:"name" binding:"required"`
	Content string `json:"content" binding:"required"`
	Type    string `json:"type"`
	Remark  string `json:"remark"`
}

type UpdateScriptRequest struct {
	Id      int    `json:"id" binding:"required"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Type    string `json:"type"`
	Remark  string `json:"remark"`
}

type PageScriptRequest struct {
	controllers.PageRequest
	Name string `json:"name"`
}
