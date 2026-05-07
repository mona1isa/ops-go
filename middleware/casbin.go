package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/models"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// CasbinMiddleware Casbin 权限控制中间件
func CasbinMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentPath := ctx.Request.URL.Path

		// 检查是否在公开路径中
		for _, path := range PublicPaths {
			if match, _ := filepath.Match(path, currentPath); match {
				ctx.Next()
				return
			}
		}

		// 获取请求接口和方法
		obj := strings.TrimRight(currentPath, "/")
		act := ctx.Request.Method

		// 获取用户信息
		userName, exists := ctx.Get("userName")
		if !exists {
			log.Println("Casbin: 未获取到用户信息")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "未授权访问",
			})
			return
		}

		username := userName.(string)

		// 获取用户详情（用于检查状态和角色）
		var user models.SysUser
		if err := models.DB.Where("user_name = ?", username).First(&user).Error; err != nil {
			log.Printf("Casbin: 查询用户信息失败: %v\n", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "用户信息无效",
			})
			return
		}

		// 检查用户是否被禁用
		if user.Status == "0" {
			log.Printf("Casbin: 用户 %s 已被禁用\n", username)
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": http.StatusForbidden,
				"msg":  "用户已被禁用，请联系管理员",
			})
			return
		}

		// 检查 Casbin 是否初始化成功
		if !models.Casbin.IsInitialized() {
			log.Printf("Casbin: 未初始化成功: %v\n", models.Casbin.GetInitError())
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"msg":  "权限系统初始化失败",
			})
			return
		}

		// 超级管理员权限检查（ID = 1 为系统预设管理员）
		if user.ID == models.AdminUserId {
			ctx.Next()
			return
		}

		// 获取用户的有效角色（状态为启用的角色）
		var userRoles []models.SysUserRole
		if err := models.DB.Where("user_id = ?", user.ID).Find(&userRoles).Error; err != nil {
			log.Printf("Casbin: 查询用户角色失败: %v\n", err)
		}

		// 检查用户是否有任何角色
		if len(userRoles) == 0 {
			log.Printf("Casbin: 用户 %s 没有分配任何角色\n", username)
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": http.StatusForbidden,
				"msg":  "无权限访问（未分配角色）",
			})
			return
		}

		// 获取用户角色ID列表
		roleIds := make([]int, 0, len(userRoles))
		for _, ur := range userRoles {
			roleIds = append(roleIds, ur.RoleId)
		}

		// 查询启用的角色
		var enabledRoles []models.SysRole
		if err := models.DB.Where("id IN ? AND status = ?", roleIds, "1").Find(&enabledRoles).Error; err != nil {
			log.Printf("Casbin: 查询角色状态失败: %v\n", err)
		}

		// 检查是否有启用的角色
		if len(enabledRoles) == 0 {
			log.Printf("Casbin: 用户 %s 的所有角色均被禁用\n", username)
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": http.StatusForbidden,
				"msg":  "无权限访问（角色已禁用）",
			})
			return
		}

		// Casbin 权限验证
		allowed, err := models.Casbin.Enforcer(username, obj, act)
		if err != nil {
			log.Printf("Casbin 权限验证失败：%v\n", err)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"msg":  "系统错误（权限验证）",
			})
			return
		}

		if !allowed {
			log.Printf("Casbin: 用户 %s 无权限访问 %s %s\n", username, act, obj)
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": http.StatusForbidden,
				"msg":  "无权限访问",
			})
			return
		}

		ctx.Next()
	}
}
