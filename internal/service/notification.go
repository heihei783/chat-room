package service

import (
	"chat-app/internal/model"
	"chat-app/internal/repository"
	"chat-app/pkg/websocket"
	"encoding/json"
	"time"
)

// NotificationService 通知业务逻辑层接口，封装通知的发送、查询与已读管理
type NotificationService interface {
	// SendNotification 发送通知：落库保存，用户在线时通过WebSocket实时推送
	SendNotification(userID uint, notification *model.Notification) error

	// GetNotifications 分页查询用户的通知列表（按时间倒序）
	GetNotifications(userID uint, page, pageSize int) ([]model.Notification, int64, error)

	// MarkAsRead 将通知标记为已读（仅限接收者本人）
	MarkAsRead(notificationID uint, userID uint) error

	// PushOfflineMessages 推送离线消息通知：将未读消息转为通知并推送（用户上线时调用）
	PushOfflineMessages(userID uint) error
}

// NotificationServiceImpl NotificationService 的默认实现
type NotificationServiceImpl struct {
	notificationRepo repository.NotificationRepository
	messageRepo      repository.MessageRepository
	hub              *websocket.Hub // WebSocket Hub，用于实时推送通知
}

// NewNotificationService 创建通知服务实例
func NewNotificationService(notificationRepo repository.NotificationRepository, messageRepo repository.MessageRepository, hub *websocket.Hub) NotificationService {
	return &NotificationServiceImpl{
		notificationRepo: notificationRepo,
		messageRepo:      messageRepo,
		hub:              hub,
	}
}

// SendNotification 发送通知：落库保存，用户在线时通过WebSocket实时推送
func (s *NotificationServiceImpl) SendNotification(userID uint, notification *model.Notification) error {
	// 1. 补充接收者与创建时间，通知落库
	notification.UserID = userID
	notification.CreatedAt = time.Now()
	if err := s.notificationRepo.Create(notification); err != nil {
		return err
	}

	// 2. 用户在线时通过WebSocket实时推送（客户端收到后即时展示）
	if s.hub != nil && s.hub.IsUserOnline(userID) {
		data := map[string]interface{}{
			"type":    "notification",
			"id":      notification.ID,
			"title":   notification.Title,
			"content": notification.Content,
			"data":    notification.Data,
			"is_read": notification.IsRead,
			"time":    notification.CreatedAt,
		}
		s.hub.SendToUser(userID, data)
	}
	return nil
}

// GetNotifications 分页查询用户的通知列表（按时间倒序）
func (s *NotificationServiceImpl) GetNotifications(userID uint, page, pageSize int) ([]model.Notification, int64, error) {
	return s.notificationRepo.GetByUser(userID, page, pageSize)
}

// MarkAsRead 将通知标记为已读（仅限接收者本人）
func (s *NotificationServiceImpl) MarkAsRead(notificationID uint, userID uint) error {
	return s.notificationRepo.MarkAsRead(notificationID, userID)
}

// PushOfflineMessages 推送离线消息通知：将未读消息转为通知并推送（用户上线时调用）
func (s *NotificationServiceImpl) PushOfflineMessages(userID uint) error {
	// 1. 获取该用户的全部未读消息
	messages, err := s.messageRepo.GetUnreadMessages(userID)
	if err != nil {
		return err
	}

	// 2. 将每条未读消息转为通知：落库并实时推送（消息对象序列化到Data字段）
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		notification := &model.Notification{
			Type:    "message",
			Title:   "新消息",
			Content: msg.Content,
			Data:    string(data),
		}
		if err := s.SendNotification(userID, notification); err != nil {
			return err
		}
	}
	return nil
}
