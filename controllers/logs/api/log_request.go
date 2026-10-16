package api

// 日志中心请求参数。所有 id 通过请求体传递，保证路由为静态路径以适配 Casbin 精确匹配。

// LogPageRequest 通用分页参数
type LogPageRequest struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

// LogSourcePageRequest 采集源分页查询
type LogSourcePageRequest struct {
	LogPageRequest
	Name       string `json:"name"`
	InstanceIp string `json:"instanceIp"`
	Path       string `json:"path"`
	Job        string `json:"job"`
	Enabled    string `json:"enabled"`
	LastStatus *int8  `json:"lastStatus"`
}

// LogSourceRequest 采集源新增/编辑
type LogSourceRequest struct {
	Id          int    `json:"id"`
	Name        string `json:"name" binding:"required"`
	InstanceId  int    `json:"instanceId" binding:"required"`
	KeyId       int    `json:"keyId"`
	Path        string `json:"path" binding:"required"`
	Job         string `json:"job"`
	Source      string `json:"source"`
	Labels      string `json:"labels"`
	LevelRegex  string `json:"levelRegex"`
	IntervalSec int    `json:"intervalSec"`
	TimeoutSec  int    `json:"timeoutSec"`
	MaxBytes    int    `json:"maxBytes"`
	BackBytes   int    `json:"backBytes"`
	Enabled     string `json:"enabled"`
	Remark      string `json:"remark"`
}

// LogIdRequest 单条记录操作（删除/详情/重置位点）
type LogIdRequest struct {
	Id int `json:"id" binding:"required"`
}

// LogToggleRequest 采集源/规则启停
type LogToggleRequest struct {
	Id      int    `json:"id" binding:"required"`
	Enabled string `json:"enabled" binding:"required"`
}

// LogQueryRequest 日志检索
type LogQueryRequest struct {
	LogPageRequest
	Selector   string `json:"selector"` // 高级用法：直接给定 LogQL 流选择器
	Job        string `json:"job"`
	Host       string `json:"host"`
	Source     string `json:"source"`
	Level      string `json:"level"`
	Keyword    string `json:"keyword"`    // 包含关键字，多个用空格分隔（AND）
	Exclude    string `json:"exclude"`    // 排除关键字，多个用空格分隔（!~）
	LineFilter string `json:"lineFilter"` // 原生 LogQL 行过滤器片段，如 |~ "foo.*bar"
	StartAt    string `json:"startAt"`
	EndAt      string `json:"endAt"`
	Limit      int    `json:"limit"`
	Direction  string `json:"direction"` // forward 正序 / backward 倒序（默认）
}

// LogHistogramRequest 日志时间分布
type LogHistogramRequest struct {
	LogQueryRequest
	Step string `json:"step"` // 直方图步长，如 1m/5m/1h
}

// LogContextRequest 命中日志的上下文，按时间取前后 N 行
type LogContextRequest struct {
	Selector  string `json:"selector"`
	Job       string `json:"job"`
	Host      string `json:"host"`
	Source    string `json:"source"`
	Path      string `json:"path"`
	Timestamp string `json:"timestamp"` // 命中行时间戳(RFC3339 或 Unix 纳秒)
	Lines     int    `json:"lines"`
}

// LogAlertPageRequest 日志告警规则分页查询
type LogAlertPageRequest struct {
	LogPageRequest
	Name    string `json:"name"`
	Level   string `json:"level"`
	Muted   string `json:"muted"`
	Enabled string `json:"enabled"`
}

// LogAlertRequest 日志告警规则新增/编辑
type LogAlertRequest struct {
	Id          int    `json:"id"`
	Name        string `json:"name" binding:"required"`
	Selector    string `json:"selector"`
	Host        string `json:"host"`
	Source      string `json:"source"`
	Keyword     string `json:"keyword"`
	Level       string `json:"level"`
	WindowSec   int    `json:"windowSec"`
	Threshold   int    `json:"threshold"`
	Severity    int8   `json:"severity"`
	CooldownSec int    `json:"cooldownSec"`
	Muted       string `json:"muted"`
	ChannelIds  string `json:"channelIds"`
	GroupByHost string `json:"groupByHost"`
	Enabled     string `json:"enabled"`
	Remark      string `json:"remark"`
}

// LogAlertSilenceRequest 规则静默
type LogAlertSilenceRequest struct {
	Id     int `json:"id" binding:"required"`
	Minute int `json:"minute" binding:"required"`
}

// LoginLogPageRequest 登录日志分页查询
type LoginLogPageRequest struct {
	LogPageRequest
	UserName string `json:"userName"`
	IpAddr   string `json:"ipAddr"`
	Result   *int8  `json:"result"`
	Platform string `json:"platform"`
	StartAt  string `json:"startAt"`
	EndAt    string `json:"endAt"`
}

// SysLogPageRequest 操作审计分页查询
type SysLogPageRequest struct {
	LogPageRequest
	CreateUser string `json:"createUser"`
	OperModule string `json:"operModule"`
	OperType   string `json:"operType"`
	Sensitive  string `json:"sensitive"`
	RequestUri string `json:"requestUri"`
	StatusCode string `json:"statusCode"`
	BizFail    string `json:"bizFail"`
	StartAt    string `json:"startAt"`
	EndAt      string `json:"endAt"`
}

// AnalyzeRequest 行为分析窗口参数
type AnalyzeRequest struct {
	Days  int `json:"days"`
	Hours int `json:"hours"`
}

// LabelsRequest 标签取值查询
type LabelsRequest struct {
	Label    string `json:"label"`
	Selector string `json:"selector"`
}
