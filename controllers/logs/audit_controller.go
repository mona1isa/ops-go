package logs

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/controllers/logs/api"
	logService "github.com/zhany/ops-go/services/logs"
)

// LoginLogController 登录日志与登录行为分析
type LoginLogController struct {
	controllers.BaseController
}

// PageHandler 登录日志分页
func (c *LoginLogController) PageHandler(ctx *gin.Context) {
	request := api.LoginLogPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	pageNum, pageSize := normalizePage(request.PageNum, request.PageSize)

	result, err := logService.PageLoginLog(pageNum, pageSize, logService.LoginLogFilter{
		UserName: request.UserName,
		IpAddr:   request.IpAddr,
		Result:   request.Result,
		Platform: request.Platform,
		StartAt:  request.StartAt,
		EndAt:    request.EndAt,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AnalyzeHandler 登录行为分析
func (c *LoginLogController) AnalyzeHandler(ctx *gin.Context) {
	request := api.AnalyzeRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		request.Days = 7
	}
	c.Success(ctx, logService.AnalyzeLogin(request.Days))
}

// AuditController 操作审计日志与行为分析
type AuditController struct {
	controllers.BaseController
}

// PageHandler 操作审计分页
func (c *AuditController) PageHandler(ctx *gin.Context) {
	request := api.SysLogPageRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数不合法："+err.Error())
		return
	}
	pageNum, pageSize := normalizePage(request.PageNum, request.PageSize)

	result, err := logService.PageSysLog(pageNum, pageSize, logService.SysLogFilter{
		CreateUser: request.CreateUser,
		OperModule: request.OperModule,
		OperType:   request.OperType,
		Sensitive:  request.Sensitive,
		RequestUri: request.RequestUri,
		StatusCode: request.StatusCode,
		BizFail:    request.BizFail,
		StartAt:    request.StartAt,
		EndAt:      request.EndAt,
	})
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	c.PageSuccess(ctx, result.Data, result.Total, result.TotalPage, result.PageNum, result.PageSize)
}

// AnalyzeHandler 操作行为分析
func (c *AuditController) AnalyzeHandler(ctx *gin.Context) {
	request := api.AnalyzeRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		request.Hours = 24
	}
	c.Success(ctx, logService.AnalyzeAudit(int64(request.Hours)))
}
