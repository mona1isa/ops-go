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
			ctx.Abort()
			return
		}

		username := userName.(string)
		if username == "admin" {
			ctx.Next()
			return
		}
		enforcer, err := models.Casbin.Enforcer(username, obj, act)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"msg":  "系统错误",
			})
			return
		}

		if !enforcer {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": http.StatusForbidden,
				"msg":  "无权限访问",
			})
		}
		ctx.Next()
	}
}
