package handler

import (
	"chat-app/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PresenceHandler 在线状态HTTP处理器：负责接收请求并调用在线状态服务层
type PresenceHandler struct {
	presenceService service.PresenceService
}

// NewPresenceHandler 创建在线状态处理器
func NewPresenceHandler(presenceService service.PresenceService) *PresenceHandler {
	return &PresenceHandler{presenceService: presenceService}
}

// GetOnlineUsers 处理获取在线用户ID列表请求：GET /api/users/online（需JWT认证）
func (h *PresenceHandler) GetOnlineUsers(c *gin.Context) {
	userIDs, err := h.presenceService.GetOnlineUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取在线用户失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userIDs,
	})
}

// GetPresence 处理查询单个用户在线状态请求：GET /api/users/:id/presence（需JWT认证）
func (h *PresenceHandler) GetPresence(c *gin.Context) {
	userID, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "用户ID无效"})
		return
	}

	status, err := h.presenceService.GetStatus(userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器内部错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id": userID,
			"status":  status,
		},
	})
}
