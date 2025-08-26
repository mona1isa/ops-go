package models

import (
	"gorm.io/gorm"
	"time"
)

type Base struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
	CreateBy  string    `gorm:"type:varchar(32);comment:创建人ID" json:"createBy"`
	UpdateBy  string    `gorm:"type:varchar(32);comment:修改人ID" json:"updateBy"`
	DelFlag   string    `gorm:"type:varchar(1);default:0;comment:删除标识（0正常 1 已删除）" json:"delFlag"`
	Remark    string    `gorm:"type:varchar(500);comment:备注" json:"remark"`
}

type PageResult[T any] struct {
	Total     int64 `json:"total"`     // 总记录数
	PageNum   int   `json:"pageNum"`   // 当前页码
	PageSize  int   `json:"pageSize"`  // 每页条数
	TotalPage int   `json:"totalPage"` // 总页数
	Data      []T   `json:"data"`      // 当前页数据
}

func Paginate[T any](db *gorm.DB, pageNum, pageSize int, scopes ...func(*gorm.DB) *gorm.DB) (PageResult[T], error) {
	var model T
	var result PageResult[T]
	result.PageNum = pageNum
	result.PageSize = pageSize

	// 应用Scope并获取总记录数
	baseQuery := db.Model(&model).Scopes(scopes...)
	if err := baseQuery.Count(&result.Total).Error; err != nil {
		return result, err
	}

	// 计算总页数
	if result.Total > 0 {
		result.TotalPage = int((result.Total + int64(pageSize) - 1) / int64(pageSize))
	}

	// 执行分页查询
	offset := (pageNum - 1) * pageSize
	if err := baseQuery.Offset(offset).Limit(pageSize).Find(&result.Data).Error; err != nil {
		return result, err
	}
	return result, nil
}
