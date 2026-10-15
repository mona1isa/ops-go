package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/monitor"
)

// MonitorRouter 主机监控告警相关路由。全部为静态 POST 路径，
// 因为 Casbin 的策略匹配是精确相等，带路径参数的路由无法授权给非超管角色。
type MonitorRouter struct{}

// Setup 注册监控模块路由
func (*MonitorRouter) Setup(r *gin.RouterGroup) {
	target := monitor.TargetController{}
	rule := monitor.RuleController{}
	event := monitor.EventController{}
	channel := monitor.ChannelController{}
	metrics := monitor.MetricsController{}

	group := r.Group("/monitor")
	{
		group.POST("/target/page", target.PageHandler)
		group.POST("/target/add", target.AddHandler)
		group.POST("/target/edit", target.EditHandler)
		group.POST("/target/delete", target.DeleteHandler)
		group.POST("/target/toggle", target.ToggleHandler)
		group.POST("/target/sync", target.SyncHandler)
		group.POST("/target/collect", target.CollectHandler)
		group.POST("/target/instance-keys", target.InstanceKeysHandler)

		group.POST("/rule/page", rule.PageHandler)
		group.POST("/rule/add", rule.AddHandler)
		group.POST("/rule/edit", rule.EditHandler)
		group.POST("/rule/delete", rule.DeleteHandler)
		group.POST("/rule/toggle", rule.ToggleHandler)
		group.POST("/rule/silence", rule.SilenceHandler)
		group.POST("/rule/test", rule.TestHandler)

		group.POST("/event/page", event.PageHandler)
		group.POST("/event/detail", event.DetailHandler)
		group.POST("/event/ack", event.AckHandler)
		group.POST("/event/delete", event.DeleteHandler)

		group.POST("/channel/page", channel.PageHandler)
		group.POST("/channel/options", channel.OptionsHandler)
		group.POST("/channel/add", channel.AddHandler)
		group.POST("/channel/edit", channel.EditHandler)
		group.POST("/channel/delete", channel.DeleteHandler)
		group.POST("/channel/toggle", channel.ToggleHandler)
		group.POST("/channel/test", channel.TestHandler)

		group.POST("/metrics/status", metrics.StatusHandler)
		group.POST("/metrics/overview", metrics.OverviewHandler)
		group.POST("/metrics/hosts", metrics.HostsHandler)
		group.POST("/metrics/instant", metrics.InstantHandler)
		group.POST("/metrics/range", metrics.RangeHandler)
	}
}
