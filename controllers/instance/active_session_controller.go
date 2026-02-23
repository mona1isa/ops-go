package instance

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/bastion"
	"github.com/zhany/ops-go/controllers"
	"net/http"
)

// ActiveSessionInfo 活跃会话信息
type ActiveSessionInfo struct {
	SessionID   string `json:"sessionId"`
	UserID      int    `json:"userId"`
	UserName    string `json:"userName"`
	InstanceID  int    `json:"instanceId"`
	InstanceIP  string `json:"instanceIp"`
	StartTime   string `json:"startTime"`
	Duration    int    `json:"duration"`
}

// ActiveSessionController 活跃会话控制器
type ActiveSessionController struct {
	controllers.BaseController
}

// ListActiveSessions 获取活跃会话列表
// @Summary 获取活跃会话列表
// @Description 获取当前所有活跃的 Bastion 会话列表（仅 Admin 可用）
// @Tags 活跃会话
// @Accept json
// @Produce json
// @Success 200 {object} controllers.Response
// @Router /api/active-sessions [get]
func (c *ActiveSessionController) List(ctx *gin.Context) {
	// 检查是否为 admin
	userInfo, exists := ctx.Get("userInfo")
	if !exists {
		c.Failure(ctx, http.StatusUnauthorized, "未授权")
		return
	}

	userMap, ok := userInfo.(map[string]interface{})
	if !ok {
		c.Failure(ctx, http.StatusInternalServerError, "用户信息格式错误")
		return
	}

	userName, _ := userMap["userName"].(string)
	if userName != "admin" {
		c.Failure(ctx, http.StatusForbidden, "仅管理员可访问")
		return
	}

	// 获取会话管理器
	manager := bastion.GetGlobalSessionManager()
	sessions := manager.ListSessions()

	// 转换为响应格式
	sessionInfos := make([]ActiveSessionInfo, 0, len(sessions))
	for _, s := range sessions {
		duration := int(s.StartTime.Sub(s.StartTime).Seconds())
		sessionInfos = append(sessionInfos, ActiveSessionInfo{
			SessionID:  s.SessionID,
			UserID:     s.UserID,
			UserName:   s.UserName,
			InstanceID: s.InstanceID,
			InstanceIP: s.InstanceIP,
			StartTime:  s.StartTime.Format("2006-01-02 15:04:05"),
			Duration:   duration,
		})
	}

	c.Success(ctx, gin.H{
		"total":    len(sessionInfos),
		"sessions": sessionInfos,
	})
}

// TerminateSession 终止会话
// @Summary 终止活跃会话
// @Description 终止指定的活跃会话（仅 Admin 可用）
// @Tags 活跃会话
// @Accept json
// @Produce json
// @Param sessionID path string true "会话ID"
// @Success 200 {object} controllers.Response
// @Router /api/active-sessions/terminate/{sessionID} [post]
func (c *ActiveSessionController) Terminate(ctx *gin.Context) {
	// 检查是否为 admin
	userInfo, exists := ctx.Get("userInfo")
	if !exists {
		c.Failure(ctx, http.StatusUnauthorized, "未授权")
		return
	}

	userMap, ok := userInfo.(map[string]interface{})
	if !ok {
		c.Failure(ctx, http.StatusInternalServerError, "用户信息格式错误")
		return
	}

	userName, _ := userMap["userName"].(string)
	if userName != "admin" {
		c.Failure(ctx, http.StatusForbidden, "仅管理员可操作")
		return
	}

	sessionID := ctx.Param("sessionID")
	if sessionID == "" {
		c.Failure(ctx, http.StatusBadRequest, "会话ID不能为空")
		return
	}

	// 获取会话管理器并终止会话
	manager := bastion.GetGlobalSessionManager()
	if err := manager.TerminateSession(sessionID); err != nil {
		if errors.Is(err, errors.New("会话不存在")) {
			c.Failure(ctx, http.StatusNotFound, err.Error())
		} else {
			c.Failure(ctx, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.Success(ctx, gin.H{
		"message": "会话已成功终止",
		"sessionID": sessionID,
	})
}
