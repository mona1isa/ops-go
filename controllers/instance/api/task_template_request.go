package api

import "github.com/zhany/ops-go/controllers"

type AddTaskTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        int8   `json:"type" binding:"required"`
	Content     string `json:"content"`
	ScriptLang  string `json:"scriptLang"`
	SrcPath     string `json:"srcPath"`
	DestPath    string `json:"destPath"`
	Timeout     int    `json:"timeout"`
	KeyId       int    `json:"keyId"`
	Description string `json:"description"`
}

type UpdateTaskTemplateRequest struct {
	Id int `json:"id" binding:"required"`
	AddTaskTemplateRequest
}

type PageTaskTemplateRequest struct {
	controllers.PageRequest
	Name string `json:"name"`
	Type *int8  `json:"type"`
}
