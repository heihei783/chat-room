package handler

import (
	"chat-app/internal/service"
	"chat-app/pkg/websocket"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var hub *websocket.Hub

// notificationService 通知服务（由main注入），用于WebSocket连接建立时推送离线通知
var notificationService service.NotificationService

// presenceService 在线状态服务（由main注入），用于WebSocket连接建立时标记用户在线
var presenceService service.PresenceService

// InitHub 初始化全局WebSocket Hub（由main调用，保证连接管理与在线状态判断共享同一实例）
func InitHub(h *websocket.Hub) {
	hub = h
	go hub.Run()
}

// InitNotificationService 注入通知服务（由main调用，用于连接建立时推送离线消息通知）
func InitNotificationService(s service.NotificationService) {
	notificationService = s
}

// InitPresenceService 注入在线状态服务（由main调用），连接建立时同步用户在线状态；
// 连接断开时的离线状态由Hub.OnUserDisconnect回调处理（在main中注入）
func InitPresenceService(s service.PresenceService) {
	presenceService = s
}

func HandleWebSocket(c *gin.Context) {
	// 获取用户ID（从JWT token中）
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未授权",
		})
		return
	}

	// 升级HTTP连接为WebSocket
	conn, err := websocket.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "WebSocket升级失败",
		})
		return
	}

	// 创建客户端
	client := &websocket.Client{
		ID:     generateClientID(),
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    hub,
	}

	// 注册客户端
	hub.Register <- client

	// 标记用户在线：写Redis缓存（TTL自动过期）+更新数据库状态+广播状态变更
	if presenceService != nil {
		_ = presenceService.SetOnline(userID)
	}

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump()

	// 推送离线未读消息通知（非关键操作，失败不影响连接）
	if notificationService != nil {
		_ = notificationService.PushOfflineMessages(userID)
	}
}

func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}
