package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从Authorization头获取token；
		// 浏览器原生WebSocket无法自定义请求头，回退支持query参数传token（如 /ws?token=xxx）
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "未提供认证token",
			})
			c.Abort()
			return
		}

		//移除“Bearer”前缀
		if len(token) > 7 && strings.HasPrefix(token, "Bearer") {
			token = token[7:]
		}

		claims, err := ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "无效的token",
			})
			c.Abort()
			return
		}

		//将用户的信息存储到上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "权限不足",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
