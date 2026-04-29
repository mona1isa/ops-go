package api

import "github.com/zhany/ops-go/controllers"

type DangerousCommandRequest struct {
	Id          int    `json:"id"`
	Name        string `json:"name" binding:"required"`
	Pattern     string `json:"pattern" binding:"required"`
	MatchType   int8   `json:"matchType"`
	Description string `json:"description"`
	IsEnabled   int8   `json:"isEnabled"`
}

type DangerousCommandPageRequest struct {
	Name string `json:"name"`
	controllers.PageRequest
}

type ChangeDangerousCommandStatusRequest struct {
	Id        int  `json:"id" binding:"required"`
	IsEnabled int8 `json:"isEnabled"`
}
