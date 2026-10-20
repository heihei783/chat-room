package model

import (
	"time"
)

// Notification 通知模型，用于站内通知（新消息提醒、系统通知等）
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"` // 接收者用户ID
	Type      string    `gorm:"not null;size:20" json:"type"`  // 通知类型：message/friend_request/system
	Title     string    `gorm:"size:100" json:"title"`         // 通知标题
	Content   string    `gorm:"size:500" json:"content"`       // 通知内容
	Data      string    `gorm:"type:text" json:"data,omitempty"` // 附加数据（JSON字符串，如关联的消息）
	IsRead    bool      `gorm:"default:false" json:"is_read"`  // 是否已读
	CreatedAt time.Time `json:"created_at"`
}

func (Notification) TableName() string {
	return "notifications"
}
