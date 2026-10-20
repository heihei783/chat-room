package repository

import (
	"chat-app/internal/model"
	"errors"

	"gorm.io/gorm"
)

// RoomRepository 聊天室数据访问层接口，抽象房间与成员相关的数据库操作
type RoomRepository interface {
	// CreateRoom 创建一个新房间
	CreateRoom(room *model.Room) error

	// GetRoomByID 根据房间ID查询房间
	GetRoomByID(id uint) (*model.Room, error)

	// GetRoomsByUser 查询指定用户加入的所有房间
	GetRoomsByUser(userID uint) ([]model.Room, error)

	// AddMember 将用户加入房间（写入房间成员表）
	AddMember(member *model.RoomMember) error

	// RemoveMember 将用户移出房间
	RemoveMember(roomID, userID uint) error

	// GetMembers 查询房间的全部成员（预加载用户信息）
	GetMembers(roomID uint) ([]model.RoomMember, error)

	// GetMemberCounts 批量统计多个房间的成员数量（返回 map：roomID → 人数）
	GetMemberCounts(roomIDs []uint) (map[uint]int64, error)

	// IsMember 判断用户是否为房间成员
	IsMember(roomID, userID uint) (bool, error)

	// GetMemberRole 查询用户在房间中的角色（owner/admin/member）
	GetMemberRole(roomID, userID uint) (string, error)
}

// RoomRepositoryImpl 基于 GORM 的 RoomRepository 实现
type RoomRepositoryImpl struct {
	db *gorm.DB
}

// NewRoomRepository 创建房间仓储实例
func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &RoomRepositoryImpl{db: db}
}

// CreateRoom 创建一个新房间
func (r *RoomRepositoryImpl) CreateRoom(room *model.Room) error {
	return r.db.Create(room).Error
}

// GetRoomByID 根据房间ID查询房间
func (r *RoomRepositoryImpl) GetRoomByID(id uint) (*model.Room, error) {
	var room model.Room
	err := r.db.First(&room, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// GetRoomsByUser 查询指定用户加入的所有房间
func (r *RoomRepositoryImpl) GetRoomsByUser(userID uint) ([]model.Room, error) {
	// 1. 查出该用户加入的所有房间ID
	var roomIDs []uint
	if err := r.db.Model(&model.RoomMember{}).
		Where("user_id = ?", userID).
		Pluck("room_id", &roomIDs).Error; err != nil {
		return nil, err
	}
	if len(roomIDs) == 0 {
		return []model.Room{}, nil
	}

	// 2. 按房间ID批量查询房间信息
	rooms := make([]model.Room, 0)
	if err := r.db.Where("id IN ?", roomIDs).Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
}

// AddMember 将用户加入房间（写入房间成员表）
func (r *RoomRepositoryImpl) AddMember(member *model.RoomMember) error {
	return r.db.Create(member).Error
}

// RemoveMember 将用户移出房间
func (r *RoomRepositoryImpl) RemoveMember(roomID, userID uint) error {
	return r.db.Where("room_id = ? AND user_id = ?", roomID, userID).
		Delete(&model.RoomMember{}).Error
}

// GetMembers 查询房间的全部成员（预加载用户信息，便于返回昵称头像等）
func (r *RoomRepositoryImpl) GetMembers(roomID uint) ([]model.RoomMember, error) {
	members := make([]model.RoomMember, 0)
	err := r.db.Preload("User").
		Where("room_id = ?", roomID).
		Find(&members).Error
	return members, err
}

// GetMemberCounts 批量统计多个房间的成员数量（GROUP BY 一次查出，避免 N+1 查询）
func (r *RoomRepositoryImpl) GetMemberCounts(roomIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(roomIDs))
	if len(roomIDs) == 0 {
		return counts, nil
	}

	type roomCount struct {
		RoomID uint
		Count  int64
	}
	rows := make([]roomCount, 0)
	if err := r.db.Model(&model.RoomMember{}).
		Select("room_id, COUNT(*) AS count").
		Where("room_id IN ?", roomIDs).
		Group("room_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		counts[row.RoomID] = row.Count
	}
	return counts, nil
}

// IsMember 判断用户是否为房间成员
func (r *RoomRepositoryImpl) IsMember(roomID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.RoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetMemberRole 查询用户在房间中的角色（owner/admin/member）
func (r *RoomRepositoryImpl) GetMemberRole(roomID, userID uint) (string, error) {
	var member model.RoomMember
	err := r.db.Where("room_id = ? AND user_id = ?", roomID, userID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return member.Role, nil
}
