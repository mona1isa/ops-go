package api

import "github.com/zhany/ops-go/controllers"

type QuickExecuteRequest struct {
	Type        int8   `json:"type" binding:"required"`
	Content     string `json:"content"`
	ScriptLang  string `json:"scriptLang"`
	SrcPath     string `json:"srcPath"`
	DestPath    string `json:"destPath"`
	InstanceIds []int  `json:"instanceIds" binding:"required"`
	KeyId       int    `json:"keyId"`
	Timeout     int    `json:"timeout"`
	Name        string `json:"name"`
}

type TemplateExecuteRequest struct {
	TemplateId  int   `json:"templateId" binding:"required"`
	InstanceIds []int `json:"instanceIds" binding:"required"`
	KeyId      int   `json:"keyId"`
	Timeout    int   `json:"timeout"`
}

type PipelineExecuteRequest struct {
	PipelineId  int   `json:"pipelineId" binding:"required"`
	InstanceIds []int `json:"instanceIds" binding:"required"`
	KeyId      int   `json:"keyId"`
}

type CancelExecutionRequest struct {
	ExecutionId uint64 `json:"executionId" binding:"required"`
}

type PageExecutionRequest struct {
	controllers.PageRequest
	Status  *int8  `json:"status"`
	Type    *int8  `json:"type"`
	StartAt string `json:"startAt"`
	EndAt   string `json:"endAt"`
}

type ExecutionDetailRequest struct {
	ExecutionId uint64 `json:"executionId" binding:"required"`
}

type HostResultRequest struct {
	ExecutionId uint64 `json:"executionId" binding:"required"`
	InstanceId  int    `json:"instanceId" binding:"required"`
}
