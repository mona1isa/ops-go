package models

import (
	"strings"
)

type SysLog struct {
	DeptId     int    `gorm:"comment:创建人部门" json:"deptId"`
	CreateUser string `gorm:"type:varchar(16);comment:创建人用户名" json:"createUser"`
	Method     string `gorm:"type:varchar(8);comment:请求方法" json:"method"`
	RequestUri string `gorm:"type:varchar(255);comment:请求地址" json:"requestUri"`
	Params     string `gorm:"type:text;comment:请求参数" json:"params"`
	Resp       string `gorm:"type:text;comment:resp" json:"resp"`
	IpAddr     string `gorm:"type:varchar(20);comment:请求IP" json:"ipAddr"`
	StatusCode string `gorm:"type:char(8);comment:响应状态码" json:"statusCode"`
	CostTimeMs int64  `gorm:"comment:响应耗时" json:"costTimeMs"`
	// 审计增强字段：接口级日志之外，补齐「操作了什么、是否敏感、业务成不成功」
	OperModule string `gorm:"column:oper_module;type:varchar(32);index;comment:操作模块" json:"operModule"`
	OperType   string `gorm:"column:oper_type;type:varchar(32);index;comment:操作类型（query/add/edit/delete/grant/other）" json:"operType"`
	Sensitive  string `gorm:"column:sensitive;type:varchar(1);default:0;index;comment:敏感操作（1 是 0 否）" json:"sensitive"`
	BizCode    string `gorm:"column:biz_code;type:varchar(16);comment:业务返回码" json:"bizCode"`
	BizMsg     string `gorm:"column:biz_msg;type:varchar(255);comment:业务返回信息" json:"bizMsg"`
	UserAgent  string `gorm:"column:user_agent;type:varchar(512);comment:客户端UA" json:"userAgent"`
	OperateAt  string `gorm:"column:operate_at;type:varchar(20);comment:操作时间(格式化)" json:"operateAt"`
	Base
}

const TableSysLog = "sys_log"

func (SysLog) TableName() string {
	return TableSysLog
}

// 敏感操作 URL 关键字：命中这些片段的写操作会额外标记留痕，供合规审计单独筛选
var sensitiveUriKeywords = []string{
	"/user/updatePassword", "/user/updatePersonalInfo", "/user/changeStatus", "/user/rm", "/user/edit",
	"/role/", "/menu/", "/dept/",
	"/keys/", "/instance-keys", "/user-instance-auth",
	"/dangerous-command", "/session-record", "/active-session",
	"/task-pipeline", "/task-execution", "/script/",
	"/monitor/channel", "/monitor/rule", "/monitor/target",
	"/cmdb/asset", "/cmdb/tag", "/cmdb/relation", "/cmdb/subnet",
}

// ClassifyOper 由请求路径推断操作模块与操作类型，并判断是否敏感操作。
// 纯字符串推断，不依赖菜单表，避免中间件引入额外查询。
func ClassifyOper(method, uri string) (module, operType string, sensitive string) {
	operType = "other"
	switch method {
	case "GET", "HEAD":
		operType = "query"
	case "POST":
		operType = postOperType(uri)
	case "PUT":
		operType = "edit"
	case "DELETE":
		operType = "delete"
	}

	module = UriModule(uri)

	if operType != "query" {
		for _, keyword := range sensitiveUriKeywords {
			if strings.Contains(uri, keyword) {
				sensitive = "1"
				break
			}
		}
	}
	if sensitive == "" {
		sensitive = "0"
	}
	return module, operType, sensitive
}

// postOperType 本项目写接口统一走 POST，按末段路径区分增删改查
func postOperType(uri string) string {
	last := uri
	if idx := strings.LastIndex(uri, "/"); idx >= 0 {
		last = uri[idx+1:]
	}
	switch last {
	case "page", "list", "detail", "info", "status", "hosts", "instant", "range", "options", "overview", "statistics", "count", "tree", "graph", "analyze", "records", "instance-keys", "available", "test", "preview", "export":
		return "query"
	case "add", "create", "save", "import", "sync", "register":
		return "add"
	case "delete", "remove", "clean", "clear", "revoke", "logout", "disable":
		return "delete"
	case "grant", "auth", "assign", "bind", "unbind", "permission":
		return "grant"
	case "edit", "update", "toggle", "enable", "reset", "password", "silence", "ack", "transition", "set-tags", "publish":
		return "edit"
	}
	return "other"
}

// UriModule 取路径中的业务段作为模块名，例如 /api/monitor/rule/page -> monitor
func UriModule(uri string) string {
	trimmed := strings.TrimPrefix(uri, "/api/")
	if idx := strings.Index(trimmed, "/"); idx > 0 {
		return trimmed[:idx]
	}
	return trimmed
}
