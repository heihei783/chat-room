package model

import (
	"time"
)

// Room 聊天室模型
type Room struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null;size:100" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	Type        string    `gorm:"default:public;size:20" json:"type"` // public, private
	Avatar      string    `gorm:"size:255" json:"avatar"`
	CreatedBy   uint      `gorm:"not null;index" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// MemberCount 成员数量：非数据库字段，由服务层查询填充（列表/详情展示用）
	MemberCount int64 `gorm:"-" json:"member_count"`

	// 关联
	Messages []Message    `gorm:"foreignKey:RoomID" json:"messages,omitempty"`
	Members  []RoomMember `gorm:"foreignKey:RoomID" json:"members,omitempty"`
	Creator  User         `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// RoomMember 聊天室成员
type RoomMember struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	RoomID uint   `gorm:"not null;index" json:"room_id"`
	UserID uint   `gorm:"not null;index" json:"user_id"`
	Role   string `gorm:"default:member;size:20" json:"role"` // owner, admin, member
	// autoCreateTime：插入时由 GORM 自动填充 time.Now()，
	// 避免零值 time.Time 被序列化为 '0000-00-00' 触发 MySQL 严格模式报错
	JoinedAt time.Time `gorm:"autoCreateTime" json:"joined_at"`

	Room Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Room) TableName() string {
	return "rooms"
}

func (RoomMember) TableName() string {
	return "room_members"
}
