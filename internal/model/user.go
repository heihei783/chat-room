package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"uniqueIndex;not null;size:50" json:"username"`
	Email     string         `gorm:"uniqueIndex;not null;size:100" json:"email"`
	Password  string         `gorm:"not null;size:255" json:"-"`
	Nickname  string         `gorm:"size:50" json:"nickname"`
	Avatar    string         `gorm:"size:255" json:"avatar"`
	Status    string         `gorm:"default:offline;size:20" json:"status"` // online, offline, away
	LastSeen  *time.Time     `json:"last_seen,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	SentMessages     []Message     `gorm:"foreignKey:SenderID" json:"-"`
	ReceivedMessages []Message     `gorm:"foreignKey:ReceiverID" json:"-"`
	RoomMembers      []RoomMember  `gorm:"foreignKey:UserID" json:"-"`
	Sessions         []UserSession `gorm:"foreignKey:UserID" json:"-"`
}

func (User) TableName() string {
	return "users"
}
