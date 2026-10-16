package models

import "time"

// 登录结果
const (
	LoginResultFail    = 0 // 失败
	LoginResultSuccess = 1 // 成功
	LoginResultLogout  = 2 // 登出
)

// 登录来源
const (
	LoginSourceWeb     = "web"     // Web 管理端
	LoginSourceBastion = "bastion" // SSH 堡垒机
)

// SysLoginLog 登录日志：记录谁、何时、从哪个 IP 登录成功或失败，满足合规审计要求
type SysLoginLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserName  string    `gorm:"column:user_name;type:varchar(64);index;comment:登录用户名" json:"userName"`
	UserId    int       `gorm:"column:user_id;type:int(11);default:0;comment:用户ID（用户不存在时为0）" json:"userId"`
	Result    int8      `gorm:"column:result;type:tinyint;default:0;index;comment:结果（0 失败 1 成功 2 登出）" json:"result"`
	Reason    string    `gorm:"column:reason;type:varchar(255);comment:失败原因" json:"reason"`
	IpAddr    string    `gorm:"column:ip_addr;type:varchar(64);index;comment:来源IP" json:"ipAddr"`
	Location  string    `gorm:"column:location;type:varchar(128);comment:来源IP归属（预留）" json:"location"`
	UserAgent string    `gorm:"column:user_agent;type:varchar(512);comment:客户端UA" json:"userAgent"`
	Browser   string    `gorm:"column:browser;type:varchar(64);comment:浏览器" json:"browser"`
	Os        string    `gorm:"column:os;type:varchar(64);comment:操作系统" json:"os"`
	Platform  string    `gorm:"column:platform;type:varchar(32);comment:登录端（web/bastion）" json:"platform"`
	DeptId    int       `gorm:"column:dept_id;type:int(11);default:0;comment:部门ID" json:"deptId"`
	CreatedAt time.Time `gorm:"column:created_at;index;comment:登录时间" json:"createdAt"`
}

const TableSysLoginLog = "sys_login_log"

func (SysLoginLog) TableName() string {
	return TableSysLoginLog
}
