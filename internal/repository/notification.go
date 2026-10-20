package repository

import (
	"chat-app/internal/model"

	"gorm.io/gorm"
)

// NotificationRepository 通知数据访问层接口，抽象通知相关的数据库操作
type NotificationRepository interface {
	// Create 保存一条新通知
	Create(notification *model.Notification) error

	// GetByUser 分页查询指定用户的通知列表（按时间倒序）
	GetByUser(userID uint, page, pageSize int) ([]model.Notification, int64, error)

	// MarkAsRead 将指定通知标记为已读（仅允许接收者本人操作）
	MarkAsRead(notificationID, userID uint) error

	// GetUnreadCount 统计指定用户的未读通知数量
	GetUnreadCount(userID uint) (int64, error)
}

// NotificationRepositoryImpl 基于 GORM 的 NotificationRepository 实现
type NotificationRepositoryImpl struct {
	db *gorm.DB
}

// NewNotificationRepository 创建通知仓储实例
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &NotificationRepositoryImpl{db: db}
}

// Create 保存一条新通知
func (r *NotificationRepositoryImpl) Create(notification *model.Notification) error {
	return r.db.Create(notification).Error
}

// GetByUser 分页查询指定用户的通知列表（按时间倒序）
func (r *NotificationRepositoryImpl) GetByUser(userID uint, page, pageSize int) ([]model.Notification, int64, error) {
	query := r.db.Model(&model.Notification{}).Where("user_id = ?", userID)

	// 统计总数（用于前端分页）
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 校验分页参数并查询当前页（最新在前）
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	notifications := make([]model.Notification, 0)
	err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&notifications).Error
	if err != nil {
		return nil, 0, err
	}
	return notifications, total, nil
}

// MarkAsRead 将指定通知标记为已读（仅允许接收者本人操作，防止越权）
func (r *NotificationRepositoryImpl) MarkAsRead(notificationID, userID uint) error {
	return r.db.Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("is_read", true).Error
}

// GetUnreadCount 统计指定用户的未读通知数量
func (r *NotificationRepositoryImpl) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}
