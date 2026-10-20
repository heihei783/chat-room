package handler

import (
	"chat-app/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SendCodeRequest 发送邮箱验证码请求参数
type SendCodeRequest struct {
	Email string `json:"email" binding:"required,email"` // 目标邮箱（必填）
}

// EmailHandler 邮件相关HTTP处理器：负责接收请求并调用邮件服务层
type EmailHandler struct {
	emailService service.EmailService
}

// NewEmailHandler 创建邮件处理器
func NewEmailHandler(emailService service.EmailService) *EmailHandler {
	return &EmailHandler{emailService: emailService}
}

// SendEmailCode 处理发送邮箱验证码请求：POST /api/auth/email-code
// 调用服务层生成验证码并发送邮件，失败时统一返回发送失败提示
func (h *EmailHandler) SendEmailCode(c *gin.Context) {
	var req SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
		})
		return
	}

	if err := h.emailService.SendVerificationCode(req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "验证码发送失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证码已发送，请查收邮箱",
	})
}

// SendResetCode 处理发送密码重置验证码请求：POST /api/auth/password/reset-code
// 不校验邮箱是否注册（统一成功返回，避免泄露注册状态），注册状态在重置时再校验
func (h *EmailHandler) SendResetCode(c *gin.Context) {
	var req SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效",
		})
		return
	}

	if err := h.emailService.SendResetCode(req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "验证码发送失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证码已发送，请查收邮箱",
	})
}
