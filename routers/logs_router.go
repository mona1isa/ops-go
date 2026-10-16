package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/logs"
)

// LogsRouter 日志中心相关路由。与监控模块一致，全部为静态 POST 路径，
// Casbin 精确匹配 RequestUrl，带路径参数的路由无法授权给非超管角色。
type LogsRouter struct{}

// Setup 注册日志中心路由
func (*LogsRouter) Setup(r *gin.RouterGroup) {
	query := logs.LogQueryController{}
	source := logs.LogSourceController{}
	alert := logs.LogAlertController{}
	login := logs.LoginLogController{}
	audit := logs.AuditController{}

	group := r.Group("/logs")
	{
		group.POST("/status", query.StatusHandler)
		group.POST("/query", query.QueryHandler)
		group.POST("/histogram", query.HistogramHandler)
		group.POST("/context", query.ContextHandler)
		group.POST("/labels", query.LabelsHandler)
		group.POST("/label-values", query.LabelValuesHandler)

		group.POST("/source/page", source.PageHandler)
		group.POST("/source/add", source.AddHandler)
		group.POST("/source/edit", source.EditHandler)
		group.POST("/source/delete", source.DeleteHandler)
		group.POST("/source/toggle", source.ToggleHandler)
		group.POST("/source/collect", source.CollectHandler)
		group.POST("/source/reset", source.ResetHandler)
		group.POST("/source/options", source.OptionsHandler)
		group.POST("/source/instance-keys", source.InstanceKeysHandler)

		group.POST("/alert/page", alert.PageHandler)
		group.POST("/alert/add", alert.AddHandler)
		group.POST("/alert/edit", alert.EditHandler)
		group.POST("/alert/delete", alert.DeleteHandler)
		group.POST("/alert/toggle", alert.ToggleHandler)
		group.POST("/alert/silence", alert.SilenceHandler)
		group.POST("/alert/test", alert.TestHandler)
		group.POST("/alert/events", alert.EventsHandler)

		group.POST("/login/page", login.PageHandler)
		group.POST("/login/analyze", login.AnalyzeHandler)

		group.POST("/audit/page", audit.PageHandler)
		group.POST("/audit/analyze", audit.AnalyzeHandler)
	}
}
