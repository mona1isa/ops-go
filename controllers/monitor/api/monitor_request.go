package api

// 监控模块请求参数。所有 id 通过请求体传递，保证路由为静态路径以适配 Casbin 精确匹配。

// MonitorPageRequest 通用分页参数
type MonitorPageRequest struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

// TargetPageRequest 采集目标分页查询
type TargetPageRequest struct {
	MonitorPageRequest
	Name       string `json:"name"`
	InstanceIp string `json:"instanceIp"`
	LastStatus *int8  `json:"lastStatus"`
	Enabled    string `json:"enabled"`
}

// TargetRequest 采集目标新增/编辑
type TargetRequest struct {
	Id          int    `json:"id"`
	InstanceId  int    `json:"instanceId" binding:"required"`
	KeyId       int    `json:"keyId"`
	IntervalSec int    `json:"intervalSec"`
	TimeoutSec  int    `json:"timeoutSec"`
	Enabled     string `json:"enabled"`
	Remark      string `json:"remark"`
}

// TargetIdRequest 采集目标删除
type TargetIdRequest struct {
	Id int `json:"id" binding:"required"`
}

// TargetInstanceKeysRequest 查询指定主机已绑定的凭证
type TargetInstanceKeysRequest struct {
	InstanceId int `json:"instanceId" binding:"required"`
}

// TargetToggleRequest 采集目标启停
type TargetToggleRequest struct {
	Id      int    `json:"id" binding:"required"`
	Enabled string `json:"enabled" binding:"required"`
}

// RulePageRequest 告警规则分页查询
type RulePageRequest struct {
	MonitorPageRequest
	Name     string `json:"name"`
	Severity *int8  `json:"severity"`
	Muted    string `json:"muted"`
	Enabled  string `json:"enabled"`
}

// RuleRequest 告警规则新增/编辑
type RuleRequest struct {
	Id          int     `json:"id"`
	Name        string  `json:"name" binding:"required"`
	Metric      string  `json:"metric"`
	PromQl      string  `json:"promQl" binding:"required"`
	Operator    string  `json:"operator"`
	Threshold   float64 `json:"threshold"`
	Severity    int8    `json:"severity"`
	ForSec      int     `json:"forSec"`
	CooldownSec int     `json:"cooldownSec"`
	Muted       string  `json:"muted"`
	ChannelIds  string  `json:"channelIds"`
	Enabled     string  `json:"enabled"`
	Remark      string  `json:"remark"`
}

// RuleIdRequest 规则删除/启停
type RuleIdRequest struct {
	Id      int    `json:"id"`
	Enabled string `json:"enabled"`
}

// RuleSilenceRequest 规则静默
type RuleSilenceRequest struct {
	Id       int   `json:"id" binding:"required"`
	Seconds  int64 `json:"seconds"`
	Disabled bool  `json:"disabled"`
}

// RuleTestRequest 规则试算
type RuleTestRequest struct {
	PromQl    string  `json:"promQl" binding:"required"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
}

// EventPageRequest 告警事件分页查询
type EventPageRequest struct {
	MonitorPageRequest
	RuleId     *int   `json:"ruleId"`
	Severity   *int8  `json:"severity"`
	Status     *int8  `json:"status"`
	InstanceIp string `json:"instanceIp"`
	Source     string `json:"source"`
	StartAt    string `json:"startAt"`
	EndAt      string `json:"endAt"`
}

// EventIdRequest 事件详情/删除
type EventIdRequest struct {
	Id uint64 `json:"id" binding:"required"`
}

// EventAckRequest 事件人工处理
type EventAckRequest struct {
	Id     uint64 `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"`
	Remark string `json:"remark"`
}

// ChannelPageRequest 通知渠道分页查询
type ChannelPageRequest struct {
	MonitorPageRequest
	Name    string `json:"name"`
	Type    *int8  `json:"type"`
	Enabled string `json:"enabled"`
}

// ChannelRequest 通知渠道新增/编辑
type ChannelRequest struct {
	Id      int    `json:"id"`
	Name    string `json:"name" binding:"required"`
	Type    int8   `json:"type"`
	Url     string `json:"url" binding:"required"`
	Secret  string `json:"secret"`
	Headers string `json:"headers"`
	Enabled string `json:"enabled"`
	Remark  string `json:"remark"`
}

// ChannelIdRequest 渠道删除/启停/测试
type ChannelIdRequest struct {
	Id      int    `json:"id"`
	Enabled string `json:"enabled"`
}

// MetricsHostRequest 主机快照查询
type MetricsHostRequest struct {
	Instance string `json:"instance" binding:"required"`
}

// MetricsRangeRequest 指标区间查询
type MetricsRangeRequest struct {
	Metric     string `json:"metric" binding:"required"`
	Instance   string `json:"instance"`
	Mountpoint string `json:"mountpoint"`
	Device     string `json:"device"`
	StartAt    int64  `json:"startAt"`
	EndAt      int64  `json:"endAt"`
	StepSec    int    `json:"stepSec"`
}
