package service

import (
	"chat-app/internal/model"
	"chat-app/internal/repository"
	"chat-app/pkg/websocket"
)

// MessageService 消息业务逻辑层接口
type MessageService interface {
	// SendMessage 发送消息并落库，群聊消息校验发送者为房间成员，随后通过WebSocket实时推送
	SendMessage(msg *model.Message) error

	// GetMessages 分页获取消息列表：私聊按双方往来查询，群聊按房间过滤（校验查询者为房间成员）
	GetMessages(userID uint, targetID *uint, roomID *uint, page, pageSize int) ([]model.Message, int64, error)

	// MarkAsRead 将指定消息标记为已读（仅限接收者本人）
	MarkAsRead(messageID uint, userID uint) error

	// GetUnreadCount 获取当前用户的未读消息数量
	GetUnreadCount(userID uint) (int64, error)
}

// MessageServiceImpl MessageService 的默认实现
type MessageServiceImpl struct {
	messageRepo repository.MessageRepository
	hub         *websocket.Hub            // WebSocket Hub，用于实时推送消息
	roomRepo    repository.RoomRepository // 房间仓储，用于群聊消息的成员权限校验
}

// NewMessageService 创建消息服务实例
func NewMessageService(messageRepo repository.MessageRepository, hub *websocket.Hub, roomRepo repository.RoomRepository) MessageService {
	return &MessageServiceImpl{
		messageRepo: messageRepo,
		hub:         hub,
		roomRepo:    roomRepo,
	}
}

// SendMessage 发送消息：校验群聊成员权限→落库→通过WebSocket实时推送
func (s *MessageServiceImpl) SendMessage(msg *model.Message) error {
	// 1. 群聊消息校验发送者是房间成员（防止任意用户向房间发消息）
	if msg.RoomID != nil {
		isMember, err := s.roomRepo.IsMember(*msg.RoomID, msg.SenderID)
		if err != nil {
			return err
		}
		if !isMember {
			return ErrNotMember
		}
	}

	// 2. 保存消息到数据库
	if err := s.messageRepo.Create(msg); err != nil {
		return err
	}

	// 3. 通过WebSocket实时推送
	s.broadcastMessage(msg)

	return nil
}

// broadcastMessage 将消息推送给接收方：私聊定向推送，群聊向房间内成员广播
func (s *MessageServiceImpl) broadcastMessage(msg *model.Message) {
	messageData := map[string]interface{}{
		"type":       "message", // 推送协议类型（message/presence/notification）
		"msg_type":   msg.Type,  // 消息类型（text/image/file）：REST 历史与 WS 实时帧字段对齐
		"id":         msg.ID,
		"sender_id":  msg.SenderID,
		"content":    msg.Content,
		"created_at": msg.CreatedAt,
	}

	// 私聊消息
	if msg.ReceiverID != nil {
		messageData["receiver_id"] = *msg.ReceiverID
		s.hub.SendToUser(*msg.ReceiverID, messageData)
	}

	// 群聊消息
	if msg.RoomID != nil {
		messageData["room_id"] = *msg.RoomID
		s.hub.BroadcastToRoom(*msg.RoomID, messageData)
	}
}

// GetMessages 分页获取消息列表：私聊按双方往来查询，群聊按房间过滤（校验查询者为房间成员）
func (s *MessageServiceImpl) GetMessages(userID uint, targetID *uint, roomID *uint, page, pageSize int) ([]model.Message, int64, error) {
	// 查询群聊消息时校验查询者是房间成员（防止窥探非加入房间的消息）
	if roomID != nil {
		isMember, err := s.roomRepo.IsMember(*roomID, userID)
		if err != nil {
			return nil, 0, err
		}
		if !isMember {
			return nil, 0, ErrNotMember
		}
	}

	return s.messageRepo.GetMessages(userID, targetID, roomID, page, pageSize)
}

// MarkAsRead 将指定消息标记为已读（仅限接收者本人）
func (s *MessageServiceImpl) MarkAsRead(messageID uint, userID uint) error {
	return s.messageRepo.MarkAsRead(messageID, userID)
}

// GetUnreadCount 获取当前用户的未读消息数量
func (s *MessageServiceImpl) GetUnreadCount(userID uint) (int64, error) {
	return s.messageRepo.GetUnreadCount(userID)
}
