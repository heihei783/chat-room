package service

import (
	"chat-app/internal/repository"
	"chat-app/pkg/websocket"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// statusCacheTTL 在线状态在Redis中的缓存有效期，过期后自动清除
const statusCacheTTL = 5 * time.Minute

// PresenceService 在线状态业务逻辑层接口
// 状态以Redis缓存为准（实时、自动过期），数据库status字段作为持久化兜底
type PresenceService interface {
	// SetOnline 将用户标记为在线（登录/建立连接时调用）
	SetOnline(userID uint) error

	// SetOffline 将用户标记为离线（登出/断开连接时调用）
	SetOffline(userID uint) error

	// SetAway 将用户标记为离开状态（away）
	SetAway(userID uint) error

	// GetStatus 获取用户当前状态（online/offline/away），优先读Redis缓存，未命中回退数据库
	GetStatus(userID uint) (string, error)

	// GetOnlineUsers 获取所有在线用户ID列表（扫描Redis状态缓存键）
	GetOnlineUsers() ([]uint, error)

	// UpdateLastSeen 更新用户的最后活跃时间（数据库）
	UpdateLastSeen(userID uint) error
}

// PresenceServiceImpl PresenceService 的默认实现
type PresenceServiceImpl struct {
	userRepo repository.UserRepository
	redis    *redis.Client // Redis缓存：在线状态实时存储
	hub      *websocket.Hub // WebSocket Hub：状态变更时广播通知
}

// NewPresenceService 创建在线状态服务实例
func NewPresenceService(userRepo repository.UserRepository, redis *redis.Client, hub *websocket.Hub) PresenceService {
	return &PresenceServiceImpl{
		userRepo: userRepo,
		redis:    redis,
		hub:      hub,
	}
}

// SetOnline 将用户标记为在线（登录/建立连接时调用）
func (s *PresenceServiceImpl) SetOnline(userID uint) error {
	return s.setStatus(userID, "online")
}

// SetOffline 将用户标记为离线（登出/断开连接时调用）
func (s *PresenceServiceImpl) SetOffline(userID uint) error {
	return s.setStatus(userID, "offline")
}

// SetAway 将用户标记为离开状态（away）
func (s *PresenceServiceImpl) SetAway(userID uint) error {
	return s.setStatus(userID, "away")
}

// setStatus 统一的状态更新流程：写数据库 → 写/删Redis缓存 → 广播状态变更
func (s *PresenceServiceImpl) setStatus(userID uint, status string) error {
	// 1. 查询用户并更新数据库中的状态与最后活跃时间
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}
	user.Status = status
	now := time.Now()
	user.LastSeen = &now
	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	// 2. 更新Redis缓存：非在线状态删除缓存键（由数据库兜底），在线状态写入并带TTL自动过期
	ctx := context.Background()
	if status == "offline" {
		if err := s.redis.Del(ctx, statusKey(userID)).Err(); err != nil {
			return err
		}
	} else {
		if err := s.redis.Set(ctx, statusKey(userID), status, statusCacheTTL).Err(); err != nil {
			return err
		}
	}

	// 3. 通过WebSocket广播状态变更（全量广播，客户端按user_id过滤）
	s.broadcastStatusChange(userID, status)

	return nil
}

// GetStatus 获取用户当前状态，优先读Redis缓存，未命中时回退数据库
func (s *PresenceServiceImpl) GetStatus(userID uint) (string, error) {
	// 1. 优先读取Redis缓存（命中即返回，性能最优）
	ctx := context.Background()
	status, err := s.redis.Get(ctx, statusKey(userID)).Result()
	if err == nil {
		return status, nil
	}
	if !errors.Is(err, redis.Nil) {
		return "", err
	}

	// 2. 缓存未命中：回退查询数据库
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrUserNotFound
		}
		return "", err
	}
	return user.Status, nil
}

// GetOnlineUsers 获取所有在线用户ID列表：扫描Redis中的状态缓存键
func (s *PresenceServiceImpl) GetOnlineUsers() ([]uint, error) {
	ctx := context.Background()
	iter := s.redis.Scan(ctx, 0, "user:status:*", 100).Iterator()

	ids := make([]uint, 0)
	for iter.Next(ctx) {
		// 键格式为 user:status:<userID>，解析出用户ID，跳过无法解析的键
		idStr := strings.TrimPrefix(iter.Val(), "user:status:")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			continue
		}
		ids = append(ids, uint(id))
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// UpdateLastSeen 更新用户的最后活跃时间（数据库）
func (s *PresenceServiceImpl) UpdateLastSeen(userID uint) error {
	return s.userRepo.UpdateLastSeen(userID, time.Now())
}

// statusKey 构建用户在Redis中的状态缓存键
func statusKey(userID uint) string {
	return fmt.Sprintf("user:status:%d", userID)
}

// broadcastStatusChange 通过WebSocket广播用户状态变更（全量广播，客户端自行按user_id过滤）
func (s *PresenceServiceImpl) broadcastStatusChange(userID uint, status string) {
	if s.hub == nil {
		return
	}

	message, err := json.Marshal(map[string]interface{}{
		"type":      "presence",
		"user_id":   userID,
		"status":    status,
		"timestamp": time.Now().Unix(),
	})
	if err != nil {
		return
	}

	select {
	case s.hub.Broadcast <- message:
	default:
		// 广播通道繁忙时丢弃本次通知，避免阻塞调用方
	}
}
