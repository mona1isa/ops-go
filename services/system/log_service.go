package system

import (
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

type LogService struct {
}

func (s *LogService) Page(request *api.LogRequest) (models.PageResult[models.SysLog], error) {
	pageNum := request.PageNum
	pageSize := request.PageSize

	var scopes []func(db *gorm.DB) *gorm.DB

	method := request.Method
	if method != "" {
		scopes = append(scopes, MethodScope(method))
	}

	uri := request.RequestURI
	if uri != "" {
		scopes = append(scopes, RequestUrlScope(uri))
	}
	pageResult, err := models.Paginate[models.SysLog](models.DB, pageNum, pageSize, scopes...)
	if err != nil {
		panic(err)
	}
	return pageResult, nil
}

func MethodScope(method string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("method = ?", method)
	}
}

func RequestUrlScope(requestUrl string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("request_uri like ?", "%"+requestUrl+"%")
	}
}
