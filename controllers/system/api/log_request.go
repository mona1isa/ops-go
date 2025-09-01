package api

import "github.com/zhany/ops-go/controllers"

type LogRequest struct {
	controllers.PageRequest
	Method     string `json:"method"`
	RequestURI string `json:"requestUri"`
}
