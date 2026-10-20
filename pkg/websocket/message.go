package websocket

import (
	"time"
)

// Message WebSocket消息结构
type Message struct {
	Type      string      `json:"type"` // 客户端发送：ping/join/leave；服务端推送：pong/message/presence/notification
	From      uint        `json:"from,omitempty"`
	To        *uint       `json:"to,omitempty"`
	RoomID    *uint       `json:"room_id,omitempty"`
	Content   string      `json:"content,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// NewMessage 创建新消息
func NewMessage(msgType string, from uint, content string) *Message {
	return &Message{
		Type:      msgType,
		From:      from,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
}
