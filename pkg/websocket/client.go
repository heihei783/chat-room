package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 连接保活与消息限制参数（由 Configure 从配置文件中读取，缺省使用以下默认值）
var (
	// pingPeriod 心跳Ping发送周期，应小于pongWait
	pingPeriod = 54 * time.Second
	// pongWait 等待Pong响应的最大时间，超过则判定连接失效
	pongWait = 60 * time.Second
	// maxMessageSize 单条消息最大字节数，防止超大消息占用内存
	maxMessageSize = int64(512)
)

// Client WebSocket客户端
type Client struct {
	ID     string
	UserID uint
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub
	mu     sync.Mutex
	// Rooms 该客户端已加入的房间集合（房间ID → 是否存在），用于群聊消息定向推送
	Rooms map[uint]bool
}

// Hub 管理所有客户端连接
type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex

	// OnUserDisconnect 用户最后一个连接断开时的回调（由业务层注入，如标记用户离线）
	// 仅在用户所有连接都断开时触发一次，避免多开场景下误标离线
	OnUserDisconnect func(userID uint)
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			h.unregister(client)
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			var slow []*Client
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					// 发送缓冲区已满：先收集慢客户端，锁外再统一注销，
					// 避免在RLock下直接close/delete造成并发读写冲突
					slow = append(slow, client)
				}
			}
			h.mu.RUnlock()

			// 慢客户端在锁外异步注销（不能在此协程内直接发送到Unregister，
			// 无缓冲通道会阻塞导致Hub协程死锁）
			if len(slow) > 0 {
				go func(clients []*Client) {
					h.mu.Lock()
					defer h.mu.Unlock()
					for _, client := range clients {
						h.unregister(client)
					}
				}(slow)
			}
		}
	}
}

// unregister 从Hub移除客户端并关闭其发送通道（调用方必须已持有写锁）
// 若该用户不再有任何在线连接，触发 OnUserDisconnect 回调（异步执行，避免阻塞Hub协程）
func (h *Hub) unregister(client *Client) {
	if _, ok := h.Clients[client]; ok {
		delete(h.Clients, client)
		close(client.Send)
		if h.OnUserDisconnect != nil && !h.hasConnection(client.UserID) {
			go h.OnUserDisconnect(client.UserID)
		}
	}
}

// hasConnection 判断指定用户是否还有其他在线连接（调用方必须已持有锁）
func (h *Hub) hasConnection(userID uint) bool {
	for client := range h.Clients {
		if client.UserID == userID {
			return true
		}
	}
	return false
}

// ReadPump 从WebSocket连接读取消息
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	// 限制单条消息大小，防止超大消息耗尽内存
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}

		// 处理接收到的消息
		c.handleMessage(message)
	}
}

// WritePump 向WebSocket连接写入消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送队列中的消息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("解析消息失败: %v", err)
		return
	}

	// 处理不同类型的消息
	switch msg.Type {
	case "ping":
		c.sendPong()
	case "join":
		// 加入房间：消息需携带room_id
		if msg.RoomID != nil {
			c.JoinRoom(*msg.RoomID)
		}
	case "leave":
		// 离开房间：消息需携带room_id
		if msg.RoomID != nil {
			c.LeaveRoom(*msg.RoomID)
		}
	default:
		// 消息发送统一走REST接口（POST /api/messages，落库后由服务层定向推送），
		// 客户端通过WebSocket仅发送控制类消息（ping/join/leave）
		log.Printf("未知消息类型: %s", msg.Type)
	}
}

func (c *Client) sendPong() {
	pong := NewMessage("pong", c.UserID, "")
	message, err := json.Marshal(pong)
	if err != nil {
		log.Printf("编码pong消息失败: %v", err)
		return
	}

	c.Send <- message
}

// JoinRoom 将客户端加入指定房间（首次加入时惰性初始化房间集合）
func (c *Client) JoinRoom(roomID uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Rooms == nil {
		c.Rooms = make(map[uint]bool)
	}
	c.Rooms[roomID] = true
}

// LeaveRoom 将客户端移出指定房间
func (c *Client) LeaveRoom(roomID uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Rooms, roomID)
}

// InRoom 判断客户端是否在指定房间中
func (c *Client) InRoom(roomID uint) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Rooms[roomID]
}

// SendToUser 向指定用户的所有在线连接发送一条JSON消息（用于私聊推送）
func (h *Hub) SendToUser(userID uint, data interface{}) {
	message, err := json.Marshal(data)
	if err != nil {
		log.Printf("编码消息失败: %v", err)
		return
	}

	h.mu.RLock()
	for client := range h.Clients {
		if client.UserID != userID {
			continue
		}
		select {
		case client.Send <- message:
		default:
			// 发送缓冲区已满：移交Run协程注销该客户端，避免阻塞并安全清理连接
			h.mu.RUnlock()
			h.Unregister <- client
			h.mu.RLock()
		}
	}
	h.mu.RUnlock()
}

// BroadcastToRoom 向指定房间内的所有在线客户端广播一条JSON消息（用于群聊推送）
func (h *Hub) BroadcastToRoom(roomID uint, data interface{}) {
	message, err := json.Marshal(data)
	if err != nil {
		log.Printf("编码消息失败: %v", err)
		return
	}

	h.mu.RLock()
	for client := range h.Clients {
		if !client.InRoom(roomID) {
			continue
		}
		select {
		case client.Send <- message:
		default:
			// 发送缓冲区已满：移交Run协程注销该客户端，避免阻塞并安全清理连接
			h.mu.RUnlock()
			h.Unregister <- client
			h.mu.RLock()
		}
	}
	h.mu.RUnlock()
}

// IsUserOnline 判断指定用户是否至少有一个在线连接（用于在线状态实时判断）
func (h *Hub) IsUserOnline(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.Clients {
		if client.UserID == userID {
			return true
		}
	}
	return false
}

// OnlineUserIDs 返回当前所有在线连接的去重用户ID列表（用于获取在线用户集合）
func (h *Hub) OnlineUserIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	seen := make(map[uint]bool)
	ids := make([]uint, 0, len(h.Clients))
	for client := range h.Clients {
		if !seen[client.UserID] {
			seen[client.UserID] = true
			ids = append(ids, client.UserID)
		}
	}
	return ids
}
