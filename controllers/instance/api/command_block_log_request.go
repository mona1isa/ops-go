package api

import (
	controllers "github.com/zhany/ops-go/controllers"
)

// CommandBlockLogPageRequest 高危指令拦截记录查询
type CommandBlockLogPageRequest struct {
	UserName     string `json:"userName"`
	RuleName     string `json:"ruleName"`
	InstanceName string `json:"instanceName"`
	SessionID    string `json:"sessionId"`
	controllers.PageRequest
}
