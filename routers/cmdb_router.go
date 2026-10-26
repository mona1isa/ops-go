package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/cmdb"
)

// CmdbRouter 资产管理相关路由。全部为静态 POST 路径，
// 因为 Casbin 的策略匹配是精确相等，带路径参数的路由无法授权给非超管角色。
type CmdbRouter struct{}

// Setup 注册资产管理模块路由
func (*CmdbRouter) Setup(r *gin.RouterGroup) {
	asset := cmdb.AssetController{}
	tag := cmdb.TagController{}
	relation := cmdb.RelationController{}
	subnet := cmdb.SubnetController{}
	certAlert := cmdb.CertAlertController{}

	group := r.Group("/cmdb")
	{
		group.POST("/asset/page", asset.PageHandler)
		group.POST("/asset/add", asset.AddHandler)
		group.POST("/asset/edit", asset.EditHandler)
		group.POST("/asset/detail", asset.DetailHandler)
		group.POST("/asset/delete", asset.DeleteHandler)
		group.POST("/asset/transition", asset.TransitionHandler)
		group.POST("/asset/import", asset.ImportHandler)
		group.POST("/asset/options", asset.OptionsHandler)
		group.POST("/asset/change-log", asset.ChangeLogHandler)

		group.POST("/tag/list", tag.ListHandler)
		group.POST("/tag/add", tag.AddHandler)
		group.POST("/tag/edit", tag.EditHandler)
		group.POST("/tag/delete", tag.DeleteHandler)
		group.POST("/tag/set", tag.SetHandler)

		group.POST("/relation/add", relation.AddHandler)
		group.POST("/relation/delete", relation.DeleteHandler)
		group.POST("/relation/list", relation.ListHandler)
		group.POST("/relation/graph-all", relation.GraphAllHandler)
		group.POST("/relation/graph", relation.GraphHandler)

		group.POST("/subnet/page", subnet.PageHandler)
		group.POST("/subnet/add", subnet.AddHandler)
		group.POST("/subnet/edit", subnet.EditHandler)
		group.POST("/subnet/delete", subnet.DeleteHandler)
		group.POST("/subnet/usage", subnet.UsageHandler)

		group.POST("/cert/rule/page", certAlert.RulePageHandler)
		group.POST("/cert/rule/add", certAlert.RuleAddHandler)
		group.POST("/cert/rule/edit", certAlert.RuleEditHandler)
		group.POST("/cert/rule/delete", certAlert.RuleDeleteHandler)
		group.POST("/cert/rule/toggle", certAlert.RuleToggleHandler)
		group.POST("/cert/rule/silence", certAlert.RuleSilenceHandler)
		group.POST("/cert/expiring", certAlert.ExpiringHandler)
		group.POST("/cert/events", certAlert.EventsHandler)
	}
}
