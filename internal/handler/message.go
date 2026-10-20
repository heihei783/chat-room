package handler

import (
	"chat-app/internal/model"
	"chat-app/internal/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MessageRequest struct {
	ReceiverID *uint  `json:"receiver_id"`
	RoomID     *uint  `json:"room_id"`
	Content    string `json:"content" binding:"required"`
	Type       string `json:"type" binding:"required,oneof=text image file"`
}

type MessageHandler struct {
	messageService service.MessageService
}

func NewMessageHandler(messageService service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
		})
		return
	}

	message := &model.Message{
		SenderID:   userID,
		ReceiverID: req.ReceiverID,
		RoomID:     req.RoomID,
		Content:    req.Content,
		Type:       req.Type,
		Status:     "sent",
	}

	if err := h.messageService.SendMessage(message); err != nil {
		h.handleMessageError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "消息发送成功",
		"data":    message,
	})
}

func (h *MessageHandler) GetMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var targetID *uint
	var roomID *uint

	if targetIDStr := c.Query("target_id"); targetIDStr != "" {
		if id, err := strconv.ParseUint(targetIDStr, 10, 32); err == nil {
			targetID = new(uint)
			*targetID = uint(id)
		}
	}

	if roomIDStr := c.Query("room_id"); roomIDStr != "" {
		if id, err := strconv.ParseUint(roomIDStr, 10, 32); err == nil {
			roomID = new(uint)
			*roomID = uint(id)
		}
	}

	messages, total, err := h.messageService.GetMessages(userID, targetID, roomID, page, pageSize)
	if err != nil {
		h.handleMessageError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     messages,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *MessageHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetUint("user_id")

	count, err := h.messageService.GetUnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取未读消息数量失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"unread_count": count,
		},
	})
}

func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || messageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "消息ID无效",
		})
		return
	}

	userID := c.GetUint("user_id")
	if err := h.messageService.MarkAsRead(uint(messageID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "标记消息已读失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "消息已标记为已读",
	})
}

// handleMessageError 将消息服务层错误映射为对应HTTP状态码并返回统一格式的错误响应
func (h *MessageHandler) handleMessageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotMember):
		// 群聊消息的成员权限校验失败：非房间成员禁止发送/查询
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "不是房间成员，无权操作"})
	default:
		// 未知错误统一返回500，避免泄露内部细节
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器内部错误"})
	}
}
