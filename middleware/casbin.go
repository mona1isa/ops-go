package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/models"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func CasbinMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentPath := ctx.Request.URL.Path
		// 检查是否在排除路径中
		for _, path := range ExcludePaths {
			if match, _ := filepath.Match(path, currentPath); match {
				ctx.Next()
				return
			}
		}

		// 获取请求接口和方法
		obj := strings.TrimRight(currentPath, "/")
		act := ctx.Request.Method

		userName, exists := ctx.Get("userName")
		if !exists {
			log.Println("未获取到用户信息")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "未授权访问",
			})
			return
		}

		username := userName.(string)
		// 超级管理员权限检查（可选：建议通过 Casbin 配置）
		if username == "admin" {
			ctx.Next()
			return
		}
		// Casbin 权限验证
		allowed, err := models.Casbin.Enforcer(username, obj, act)
		if err != nil {
			log.Printf("Casbin 权限验证失败：%v\n", err)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"msg":  "系统错误",
			})
			return
		}

		if !allowed {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": http.StatusForbidden,
				"msg":  "无权限访问",
			})
			return
		}
		ctx.Next()
	}
}
