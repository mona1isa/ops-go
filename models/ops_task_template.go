package models

// 任务类型
const (
	TaskTypeCommand = 1 // 执行命令
	TaskTypeScript  = 2 // 执行脚本
	TaskTypeFile    = 3 // 分发文件
)

// OpsTaskTemplate 任务模板
type OpsTaskTemplate struct {
	Name        string `gorm:"type:varchar(128);not null;comment:模板名称" json:"name"`
	Type        int8   `gorm:"type:tinyint;not null;comment:任务类型 1执行命令 2执行脚本 3分发文件" json:"type"`
	Content     string `gorm:"type:text;comment:命令内容/脚本内容" json:"content"`
	ScriptLang  string `gorm:"type:varchar(32);comment:脚本语言 shell/python" json:"scriptLang"`
	SrcPath     string `gorm:"type:varchar(512);comment:源文件路径" json:"srcPath"`
	DestPath    string `gorm:"type:varchar(512);comment:目标路径" json:"destPath"`
	Timeout     int    `gorm:"type:int;default:300;comment:超时时间(秒)" json:"timeout"`
	KeyId       int    `gorm:"type:int;default:0;comment:凭证ID 0自动选择" json:"keyId"`
	Description string `gorm:"type:varchar(255);comment:描述" json:"description"`
	Base
}
