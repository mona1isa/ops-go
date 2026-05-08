package api

import "github.com/zhany/ops-go/controllers"

type PipelineStepInput struct {
	StepName     string `json:"stepName" binding:"required"`
	TemplateId   int    `json:"templateId" binding:"required"`
	StepOrder    int    `json:"stepOrder"`
	ParentStepId int    `json:"parentStepId"`
	OnFailure   int8   `json:"onFailure"`
	RetryCount   int    `json:"retryCount"`
}

type AddPipelineRequest struct {
	Name        string              `json:"name" binding:"required"`
	Description string              `json:"description"`
	Steps       []PipelineStepInput `json:"steps" binding:"required"`
}

type UpdatePipelineRequest struct {
	Id int `json:"id" binding:"required"`
	AddPipelineRequest
}

type PagePipelineRequest struct {
	controllers.PageRequest
	Name string `json:"name"`
}
