package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

func CorsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		// 允许的源（域名或IP地址），开发阶段可以放行全部，生产环境应严格限制
		//AllowOrigins: []string{"http://localhost:8080", "http://192.168.124.22:8080", "https://ops-go.com"},
		AllowOrigins: []string{"*"},
		// 允许的 HTTP 方法
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		// 允许的请求头
		AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Forwarded-For"},
		// 允许暴露给前端的响应头
		ExposeHeaders: []string{"Content-Length"},
		// 是否允许携带认证信息（如 cookies）
		AllowCredentials: true,
		// 预检请求的缓存时间，单位为秒
		MaxAge: 12 * time.Hour,
	})
}
