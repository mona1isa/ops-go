package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

type BaseController struct {
}

type PageRequest struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

type PageData struct {
	Data      any   `json:"data"`      // 当前页数据
	Total     int64 `json:"total"`     //	总记录数
	TotalPage int   `json:"totalPage"` //	总页数
	PageNum   int   `json:"pageNum"`   //	当前页码
	PageSize  int   `json:"pageSize"`  //	每页大小
}

func (b *BaseController) JustSuccess(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "success",
	})
}

func (b *BaseController) Success(ctx *gin.Context, data any) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "success",
		"data": data,
	})
}

func (b *BaseController) Failure(ctx *gin.Context, code int, msg any) {
	ctx.JSON(code, gin.H{
		"code": code,
		"msg":  msg,
	})
}

func (b *BaseController) PageSuccess(ctx *gin.Context, data any, total int64, totalPage, pageNum, pageSize int) {
	ctx.JSON(http.StatusOK, gin.H{
		"code":      http.StatusOK,
		"msg":       "success",
		"data":      data,
		"total":     total,
		"totalPage": totalPage,
		"pageNum":   pageNum,
		"pageSize":  pageSize,
	})
}

func (b *BaseController) PageParams(ctx *gin.Context) (*PageRequest, error) {
	pageRequest := PageRequest{}
	err := ctx.ShouldBindJSON(&pageRequest)
	if err != nil {
		return nil, errors.New("获取分页参数异常")
	}
	return &pageRequest, nil

}

func (b *BaseController) GetUserId(ctx *gin.Context) string {
	valueUserId, exists := ctx.Get("userId")
	if !exists {
		return ""
	}
	return valueUserId.(string)
}

func (b *BaseController) GetDeptId(ctx *gin.Context) string {
	valueDeptId, exists := ctx.Get("deptId")
	if !exists {
		return ""
	}
	return valueDeptId.(string)
}

func (b *BaseController) GetUserName(ctx *gin.Context) string {
	valueUserName, exists := ctx.Get("userName")
	if !exists {
		return ""
	}
	return valueUserName.(string)
}
