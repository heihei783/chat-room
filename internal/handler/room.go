package handler

import (
	"chat-app/internal/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateRoomRequest 创建房间请求参数
type CreateRoomRequest struct {
	Name        string `json:"name" binding:"required"` // 房间名称（必填）
	Description string `json:"description"`             // 房间描述（选填）
	Type        string `json:"type"`                    // 房间类型：public/private，缺省public
	Avatar      string `json:"avatar"`                  // 房间头像URL（选填）
}

// RoomHandler 聊天室HTTP处理器：负责参数校验、调用服务层并返回统一格式的响应
type RoomHandler struct {
	roomService service.RoomService
}

// NewRoomHandler 创建聊天室处理器
func NewRoomHandler(roomService service.RoomService) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

// CreateRoom 处理创建房间请求：POST /api/rooms（需JWT认证）
func (h *RoomHandler) CreateRoom(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
		})
		return
	}

	room, err := h.roomService.CreateRoom(req.Name, req.Description, req.Type, req.Avatar, userID)
	if err != nil {
		h.handleRoomError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "房间创建成功",
		"data":    room,
	})
}

// GetUserRooms 处理查询当前用户加入的房间列表：GET /api/rooms（需JWT认证）
func (h *RoomHandler) GetUserRooms(c *gin.Context) {
	userID := c.GetUint("user_id")

	rooms, err := h.roomService.GetUserRooms(userID)
	if err != nil {
		h.handleRoomError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rooms,
	})
}

// GetRoom 处理查询房间详情：GET /api/rooms/:id（需JWT认证）
func (h *RoomHandler) GetRoom(c *gin.Context) {
	roomID, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "房间ID无效"})
		return
	}

	room, err := h.roomService.GetRoom(roomID)
	if err != nil {
		h.handleRoomError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    room,
	})
}

// JoinRoom 处理加入房间请求：POST /api/rooms/:id/join（需JWT认证）
func (h *RoomHandler) JoinRoom(c *gin.Context) {
	userID := c.GetUint("user_id")

	roomID, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "房间ID无效"})
		return
	}

	if err := h.roomService.JoinRoom(roomID, userID); err != nil {
		h.handleRoomError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已加入房间",
	})
}

// LeaveRoom 处理退出房间请求：POST /api/rooms/:id/leave（需JWT认证）
func (h *RoomHandler) LeaveRoom(c *gin.Context) {
	userID := c.GetUint("user_id")

	roomID, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "房间ID无效"})
		return
	}

	if err := h.roomService.LeaveRoom(roomID, userID); err != nil {
		h.handleRoomError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已退出房间",
	})
}

// GetMembers 处理查询房间成员列表：GET /api/rooms/:id/members（需JWT认证）
func (h *RoomHandler) GetMembers(c *gin.Context) {
	roomID, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "房间ID无效"})
		return
	}

	members, err := h.roomService.GetMembers(roomID)
	if err != nil {
		h.handleRoomError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
	})
}

// parseUintParam 解析路径中的正整数参数（如房间ID、用户ID）
func parseUintParam(c *gin.Context, key string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(key), 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

// handleRoomError 将房间服务层错误映射为对应HTTP状态码并返回统一格式的错误响应
func (h *RoomHandler) handleRoomError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrRoomNameEmpty):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "房间名称不能为空"})
	case errors.Is(err, service.ErrRoomNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "房间不存在"})
	case errors.Is(err, service.ErrAlreadyMember):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "已经是房间成员"})
	case errors.Is(err, service.ErrNotMember):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "不是房间成员"})
	case errors.Is(err, service.ErrOwnerCannotLeave):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "群主不能退出房间"})
	default:
		// 未知错误统一返回500，避免泄露内部细节
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器内部错误"})
	}
}
