package model

import (
	"time"
)

// Message 消息模型
type Message struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Type      string    `gorm:"default:text;size:20" json:"type"`   // text, image, file
	Status    string    `gorm:"default:sent;size:20" json:"status"` // sent, delivered, read
	CreatedAt time.Time `json:"created_at"`

	// 外键
	SenderID   uint  `gorm:"not null;index" json:"sender_id"`
	ReceiverID *uint `gorm:"index" json:"receiver_id,omitempty"` // 私聊接收者
	RoomID     *uint `gorm:"index" json:"room_id,omitempty"`     // 群聊房间

	// 关联
	Sender   User  `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Receiver *User `gorm:"foreignKey:ReceiverID" json:"receiver,omitempty"`
	Room     *Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

func (Message) TableName() string {
	return "messages"
}
