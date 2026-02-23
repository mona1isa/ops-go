package instance

import (
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers"
	"github.com/zhany/ops-go/services/instance"
	"net/http"
	"strconv"
)

type SessionRecordController struct {
	controllers.BaseController
}

// List 获取会话记录列表
// @Summary 获取会话记录列表
// @Tags 会话记录
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param userId query int false "用户ID"
// @Param instanceId query int false "主机ID"
// @Param status query int false "状态"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} controllers.Response
// @Router /api/session-record/list [get]
func (c *SessionRecordController) List(ctx *gin.Context) {
	var req instance.ListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		c.Failure(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	service := &instance.SessionRecordService{}
	resp, err := service.List(&req)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	c.Success(ctx, resp)
}

// Get 获取会话记录详情
// @Summary 获取会话记录详情
// @Tags 会话记录
// @Accept json
// @Produce json
// @Param id path int true "会话记录ID"
// @Success 200 {object} controllers.Response
// @Router /api/session-record/{id} [get]
func (c *SessionRecordController) Get(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "ID格式错误")
		return
	}

	service := &instance.SessionRecordService{}
	record, err := service.GetByID(id)
	if err != nil {
		c.Failure(ctx, http.StatusNotFound, err.Error())
		return
	}

	c.Success(ctx, record)
}

// Delete 删除会话记录
// @Summary 删除会话记录
// @Tags 会话记录
// @Accept json
// @Produce json
// @Param id path int true "会话记录ID"
// @Success 200 {object} controllers.Response
// @Router /api/session-record/{id} [delete]
func (c *SessionRecordController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "ID格式错误")
		return
	}

	service := &instance.SessionRecordService{}
	if err := service.Delete(id); err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	c.Success(ctx, nil)
}

// Playback 回放会话
// @Summary 回放会话
// @Description 返回录像内容供前端播放
// @Tags 会话记录
// @Accept json
// @Produce json
// @Param id path int true "会话记录ID"
// @Success 200 {object} controllers.Response
// @Router /api/session-record/playback/{id} [get]
func (c *SessionRecordController) Playback(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Failure(ctx, http.StatusBadRequest, "ID格式错误")
		return
	}

	service := &instance.SessionRecordService{}
	record, err := service.GetByID(id)
	if err != nil {
		c.Failure(ctx, http.StatusNotFound, err.Error())
		return
	}

	if record.RecordingFile == "" {
		c.Failure(ctx, http.StatusNotFound, "录像文件不存在")
		return
	}

	// 读取录像文件内容
	content, err := service.GetPlaybackContent(record.RecordingFile)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "读取录像文件失败: "+err.Error())
		return
	}

	c.Success(ctx, map[string]interface{}{
		"content": content,
	})
}

// Statistics 获取统计数据
// @Summary 获取统计数据
// @Tags 会话记录
// @Accept json
// @Produce json
// @Param userId query int false "用户ID"
// @Success 200 {object} controllers.Response
// @Router /api/session-record/statistics [get]
func (c *SessionRecordController) Statistics(ctx *gin.Context) {
	userIdStr := ctx.Query("userId")
	userId, _ := strconv.Atoi(userIdStr)

	service := &instance.SessionRecordService{}
	stats, err := service.GetStatistics(userId)
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "查询统计数据失败: "+err.Error())
		return
	}

	c.Success(ctx, stats)
}

// Download 下载录像文件
// @Summary 下载录像文件
// @Tags 会话记录
// @Accept json
// @Produce octet-stream
// @Param id path int true "会话记录ID"
// @Success 200 {file} file
// @Router /api/session-record/download/{id} [get]
func (c *SessionRecordController) Download(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": http.StatusBadRequest,
			"msg":  "ID格式错误",
		})
		return
	}

	service := &instance.SessionRecordService{}
	record, err := service.GetByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"code": http.StatusNotFound,
			"msg":  err.Error(),
		})
		return
	}

	if record.RecordingFile == "" {
		ctx.JSON(http.StatusNotFound, gin.H{
			"code": http.StatusNotFound,
			"msg":  "录像文件不存在",
		})
		return
	}

	// 下载文件
	ctx.FileAttachment(record.RecordingFile, record.SessionID+".cast")
}

// ListActiveSessions 获取活跃会话列表
// @Summary 获取活跃会话列表
// @Tags 会话记录
// @Accept json
// @Produce json
// @Success 200 {object} controllers.Response
// @Router /api/session-record/active [get]
func (c *SessionRecordController) ListActiveSessions(ctx *gin.Context) {
	service := &instance.SessionRecordService{}
	sessions, err := service.ListActiveSessions()
	if err != nil {
		c.Failure(ctx, http.StatusInternalServerError, "获取活跃会话失败: "+err.Error())
		return
	}

	c.Success(ctx, sessions)
}

// TerminateSession 终止会话
// @Summary 终止会话
// @Tags 会话记录
// @Accept json
// @Produce json
// @Param sessionID path string true "会话ID"
// @Success 200 {object} controllers.Response
// @Router /api/session-record/terminate/{sessionID} [post]
func (c *SessionRecordController) TerminateSession(ctx *gin.Context) {
	sessionID := ctx.Param("sessionID")

	// 简化权限校验：只检查是否为 admin
	// 从上下文获取用户名，中间件已经验证了登录状态
	userName, exists := ctx.Get("userName")
	if !exists {
		c.Failure(ctx, http.StatusBadRequest, "用户信息不存在")
		return
	}

	// 判断是否为 admin
	userNameStr, ok := userName.(string)
	if !ok || userNameStr != "admin" {
		c.Failure(ctx, http.StatusForbidden, "只有管理员可以终止会话")
		return
	}

	// admin 可以终止任意会话，不需要检查 userId
	service := &instance.SessionRecordService{}
	if err := service.TerminateSession(sessionID, true, 0); err != nil {
		c.Failure(ctx, http.StatusBadRequest, err.Error())
		return
	}

	c.Success(ctx, gin.H{
		"message": "会话已终止",
	})
}
