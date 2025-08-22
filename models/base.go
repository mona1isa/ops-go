package models

import (
	"time"
)

type Base struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
	CreateBy  string    `gorm:"type:varchar(32);comment:创建人ID" json:"createBy"`
	UpdateBy  string    `gorm:"type:varchar(32);comment:修改人ID" json:"updateBy"`
	DelFlag   string    `gorm:"type:varchar(1);comment:删除标识（0正常 1 已删除）" json:"delFlag"`
	Remark    string    `gorm:"type:varchar(500);comment:备注" json:"remark"`
}
