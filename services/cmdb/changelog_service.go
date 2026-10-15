package cmdb

import (
	"errors"
	"log"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// ChangeLogService 资产变更记录查询
type ChangeLogService struct{}

// writeChangeLog 写入一条资产变更日志，detail 按字段长度截断
func writeChangeLog(assetId int, action, detail, userId string) {
	logRow := models.OpsCmdbChangeLog{
		AssetId: assetId,
		Action:  action,
		Detail:  truncate(detail, 1000),
		Base:    models.Base{CreateBy: userId, UpdateBy: userId},
	}
	if err := models.DB.Create(&logRow).Error; err != nil {
		log.Printf("写入资产变更日志失败 assetId=%d: %v", assetId, err)
	}
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// List 分页查询某资产的变更记录
func (s *ChangeLogService) List(assetId, pageNum, pageSize int) (models.PageResult[models.OpsCmdbChangeLog], error) {
	result, err := models.Paginate[models.OpsCmdbChangeLog](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		return db.Where("asset_id = ?", assetId).Order("id DESC")
	})
	if err != nil {
		log.Println("查询资产变更记录失败:", err)
		return result, errors.New("查询变更记录失败")
	}
	return result, nil
}
