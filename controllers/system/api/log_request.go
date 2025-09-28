package api

import "github.com/zhany/ops-go/controllers"

type LogRequest struct {
	controllers.PageRequest
	CreateUser string `json:"createUser"`
	Method     string `json:"method"`
	RequestURI string `json:"requestUri"`
	StatusCode string `json:"statusCode"`
}
