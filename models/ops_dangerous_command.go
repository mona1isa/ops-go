package models

// OpsDangerousCommand 高危指令规则表
type OpsDangerousCommand struct {
	Name        string `gorm:"type:varchar(64);not null;comment:规则名称" json:"name"`
	Pattern     string `gorm:"type:varchar(255);not null;comment:匹配模式" json:"pattern"`
	MatchType   int8   `gorm:"type:tinyint;default:1;comment:匹配类型 1精确匹配 2前缀匹配 3正则匹配" json:"matchType"`
	Description string `gorm:"type:varchar(255);comment:描述说明" json:"description"`
	IsEnabled   int8   `gorm:"type:tinyint;default:1;comment:是否启用 0禁用 1启用" json:"isEnabled"`
	IsBuiltin   int8   `gorm:"type:tinyint;default:0;comment:是否内置 0自定义 1内置" json:"isBuiltin"`
	Base
}

const TableOpsDangerousCommand = "ops_dangerous_command"

func (OpsDangerousCommand) TableName() string {
	return TableOpsDangerousCommand
}

// MatchType 常量定义
const (
	MatchTypeExact  int8 = 1 // 精确匹配
	MatchTypePrefix int8 = 2 // 前缀匹配
	MatchTypeRegex  int8 = 3 // 正则匹配
)
