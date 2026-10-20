package repository

import (
	"chat-app/internal/model"

	"gorm.io/gorm"
)

// MessageRepository 消息数据访问层接口，抽象消息相关的数据库操作
type MessageRepository interface {
	// Create 保存一条新消息（发送消息时落库）
	Create(msg *model.Message) error

	// GetMessages 分页查询消息列表：
	// 传入roomID时查群聊消息；否则传入targetID时查双方私聊往来；两者都不传时查当前用户相关消息
	GetMessages(userID uint, targetID *uint, roomID *uint, page, pageSize int) ([]model.Message, int64, error)

	// MarkAsRead 将指定消息标记为已读（仅允许接收者本人操作）
	MarkAsRead(messageID uint, userID uint) error

	// GetUnreadCount 统计当前用户的未读消息数量
	GetUnreadCount(userID uint) (int64, error)

	// GetUnreadMessages 查询当前用户的全部未读消息（用于上线时推送离线通知）
	GetUnreadMessages(userID uint) ([]model.Message, error)
}

// MessageRepositoryImpl 基于 GORM 的 MessageRepository 实现
type MessageRepositoryImpl struct {
	db *gorm.DB
}

// NewMessageRepository 创建消息仓储实例
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &MessageRepositoryImpl{db: db}
}

// Create 保存一条新消息（发送消息时落库）
func (r *MessageRepositoryImpl) Create(msg *model.Message) error {
	return r.db.Create(msg).Error
}

// GetMessages 分页查询消息列表：
// 群聊按房间过滤；私聊查双方往来（双向）；未指定会话时查当前用户发送或接收的全部消息
func (r *MessageRepositoryImpl) GetMessages(userID uint, targetID *uint, roomID *uint, page, pageSize int) ([]model.Message, int64, error) {
	// 1. 根据会话类型构建过滤条件
	query := r.db.Model(&model.Message{})

	switch {
	case roomID != nil:
		// 群聊消息：按房间ID过滤
		query = query.Where("room_id = ?", *roomID)
	case targetID != nil:
		// 私聊消息：查询当前用户与对方的双向往来
		query = query.Where(
			"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
			userID, *targetID, *targetID, userID,
		)
	default:
		// 未指定会话：查询当前用户发送或接收的所有消息
		query = query.Where("sender_id = ? OR receiver_id = ?", userID, userID)
	}

	// 2. 统计符合条件的消息总数（用于前端分页）
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 3. 校验分页参数，查询当前页数据（按时间倒序，最新在前）
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	messages := make([]model.Message, 0)
	err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&messages).Error
	if err != nil {
		return nil, 0, err
	}
	return messages, total, nil
}

// MarkAsRead 将指定消息标记为已读（仅允许接收者本人操作，防止越权）
func (r *MessageRepositoryImpl) MarkAsRead(messageID uint, userID uint) error {
	return r.db.Model(&model.Message{}).
		Where("id = ? AND receiver_id = ?", messageID, userID).
		Update("status", "read").Error
}

// GetUnreadCount 统计当前用户的未读消息数量（以接收者身份，状态非read的私聊消息）
func (r *MessageRepositoryImpl) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Message{}).
		Where("receiver_id = ? AND status <> ?", userID, "read").
		Count(&count).Error
	return count, err
}

// GetUnreadMessages 查询当前用户的全部未读消息（按时间正序，用于上线时推送离线通知）
func (r *MessageRepositoryImpl) GetUnreadMessages(userID uint) ([]model.Message, error) {
	messages := make([]model.Message, 0)
	err := r.db.Where("receiver_id = ? AND status <> ?", userID, "read").
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}
