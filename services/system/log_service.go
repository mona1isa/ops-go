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

	createUser := request.CreateUser
	if createUser != "" {
		scopes = append(scopes, CreateUserScope(createUser))
	}

	statusCode := request.StatusCode
	if statusCode != "" {
		scopes = append(scopes, StatusCodeScope(statusCode))
	}
	// 按 id 降序排列
	scopes = append(scopes, func(db *gorm.DB) *gorm.DB {
		return db.Order("id desc")
	})

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

func CreateUserScope(createUser string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("create_user = ?", createUser)
	}
}

func StatusCodeScope(statusCode string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status_code = ?", statusCode)
	}
}
