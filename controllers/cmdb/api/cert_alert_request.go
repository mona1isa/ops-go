package api

// 证书/域名到期告警请求参数。id 一律走请求体，保证路由为静态路径以适配 Casbin 精确匹配。

// CertRulePageRequest 证书告警规则分页查询
type CertRulePageRequest struct {
	CmdbPageRequest
	Name    string `json:"name"`
	Enabled string `json:"enabled"`
	Muted   string `json:"muted"`
}

// CertRuleRequest 证书告警规则新增/编辑
type CertRuleRequest struct {
	Id          int    `json:"id"`
	Name        string `json:"name" binding:"required"`
	ModelTypes  string `json:"modelTypes"`
	WarningDays int    `json:"warningDays"`
	Severity    int8   `json:"severity"`
	CooldownSec int    `json:"cooldownSec"`
	Muted       string `json:"muted"`
	ChannelIds  string `json:"channelIds"`
	Enabled     string `json:"enabled"`
	Remark      string `json:"remark"`
}

// CertRuleIdRequest 规则删除
type CertRuleIdRequest struct {
	Id int `json:"id" binding:"required"`
}

// CertRuleToggleRequest 规则启停
type CertRuleToggleRequest struct {
	Id      int    `json:"id" binding:"required"`
	Enabled string `json:"enabled" binding:"required"`
}

// CertRuleSilenceRequest 规则静默
type CertRuleSilenceRequest struct {
	Id     int `json:"id" binding:"required"`
	Minute int `json:"minute"`
}

// CertExpiringRequest 到期概览查询
type CertExpiringRequest struct {
	ModelTypes  string `json:"modelTypes"`
	WarningDays int    `json:"warningDays"`
}
