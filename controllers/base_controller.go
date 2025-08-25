package controllers

import (
	"github.com/gin-gonic/gin"
)

type BaseController struct {
}

type UserInfo struct {
	UserId   int    `json:"userId"`
	DeptId   int    `json:"deptId"`
	UserName string `json:"userName"`
}

func (b *BaseController) GetUserInfo(ctx *gin.Context) *UserInfo {

	return nil
}
