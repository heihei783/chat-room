package handler

import (
	"chat-app/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRequest 用户注册请求参数
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`       // 用户名（必填）
	Email    string `json:"email" binding:"required,email"`    // 邮箱（必填，注册依据）
	Password string `json:"password" binding:"required,min=6"` // 密码（必填，至少6位）
	Code     string `json:"code" binding:"required,len=6"`     // 邮箱验证码（必填，6位数字）
	Nickname string `json:"nickname"`                          // 昵称（选填）
}

// LoginRequest 用户登录请求参数
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"` // 邮箱（必填）
	Password string `json:"password" binding:"required"`    // 密码（必填）
}

// UpdateProfileRequest 更新资料请求参数
type UpdateProfileRequest struct {
	Nickname string `json:"nickname"` // 新昵称（选填）
	Avatar   string `json:"avatar"`   // 新头像URL（选填）
}

// ResetPasswordRequest 重置密码请求参数
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`           // 邮箱（必填）
	Code        string `json:"code" binding:"required,len=6"`            // 邮箱验证码（必填，6位数字）
	NewPassword string `json:"new_password" binding:"required,min=6"`    // 新密码（必填，至少6位）
}

// UserHandler 用户HTTP处理器：负责参数校验、调用服务层并返回统一格式的响应
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Register 处理用户注册请求：POST /api/auth/register
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
		})
		return
	}

	// 调用服务层执行注册（含验证码校验）
	user, err := h.userService.Register(req.Username, req.Email, req.Password, req.Nickname, req.Code)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "注册成功",
		"data":    user,
	})
}

// Login 处理用户登录请求：POST /api/auth/login，成功后返回JWT令牌与用户信息
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
		})
		return
	}

	// 获取客户端IP与User-Agent，用于会话记录
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	result, err := h.userService.Login(req.Email, req.Password, ip, userAgent)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "登录成功",
		"data": gin.H{
			"token": result.Token,
			"user":  result.User,
		},
	})
}

// GetProfile 获取当前登录用户资料：GET /api/user/profile（需JWT认证）
func (h *UserHandler) GetProfile(c *gin.Context) {
	// 用户ID由认证中间件从JWT解析后注入上下文
	userID := c.GetUint("user_id")

	user, err := h.userService.GetProfile(userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// UpdateProfile 更新当前登录用户资料：PUT /api/user/profile（需JWT认证）
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
		})
		return
	}

	// 至少需要提供一个待更新字段
	if req.Nickname == "" && req.Avatar == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请提供需要更新的内容",
		})
		return
	}

	if err := h.userService.UpdateProfile(userID, req.Nickname, req.Avatar); err != nil {
		h.handleServiceError(c, err)
		return
	}

	// 返回更新后的最新资料
	user, err := h.userService.GetProfile(userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "资料更新成功",
		"data":    user,
	})
}

// ResetPassword 处理重置密码请求：POST /api/auth/password/reset
// 校验邮箱验证码后更新为新密码，成功后需用新密码重新登录
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
		})
		return
	}

	if err := h.userService.ResetPassword(req.Email, req.Code, req.NewPassword); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "密码重置成功，请使用新密码登录",
	})
}

// handleServiceError 将服务层错误映射为对应HTTP状态码并返回统一格式的错误响应
func (h *UserHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEmailExists):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "邮箱已被注册"})
	case errors.Is(err, service.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
	case errors.Is(err, service.ErrInvalidPassword):
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "邮箱或密码错误"})
	case errors.Is(err, service.ErrWeakPassword):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "密码长度不能少于6位"})
	case errors.Is(err, service.ErrCodeNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "验证码不存在或已失效，请重新获取"})
	case errors.Is(err, service.ErrCodeMismatch):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "验证码错误"})
	default:
		// 未知错误统一返回500，避免泄露内部细节
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器内部错误"})
	}
}
