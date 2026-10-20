package service

import (
	"chat-app/internal/model"
	"chat-app/internal/repository"
	"errors"
	"strings"
)

// 房间服务错误定义：供上层(handler)区分错误类型并映射HTTP状态码
var (
	// ErrRoomNameEmpty 房间名称不能为空
	ErrRoomNameEmpty = errors.New("房间名称不能为空")
	// ErrRoomNotFound 房间不存在
	ErrRoomNotFound = errors.New("房间不存在")
	// ErrAlreadyMember 用户已是房间成员
	ErrAlreadyMember = errors.New("已经是房间成员")
	// ErrNotMember 用户不是房间成员
	ErrNotMember = errors.New("不是房间成员")
	// ErrOwnerCannotLeave 群主不能退出房间
	ErrOwnerCannotLeave = errors.New("群主不能退出房间")
)

// RoomService 聊天室业务逻辑层接口，封装房间的创建、加入、退出与查询
type RoomService interface {
	// CreateRoom 创建房间，创建者自动成为群主成员（role=owner）
	CreateRoom(name, description, roomType, avatar string, ownerID uint) (*model.Room, error)

	// JoinRoom 用户加入房间（已是成员时返回 ErrAlreadyMember）
	JoinRoom(roomID, userID uint) error

	// LeaveRoom 用户退出房间（非成员或群主退出时返回错误）
	LeaveRoom(roomID, userID uint) error

	// GetRoom 查询房间基本信息
	GetRoom(roomID uint) (*model.Room, error)

	// GetUserRooms 查询用户加入的所有房间
	GetUserRooms(userID uint) ([]model.Room, error)

	// GetMembers 查询房间的全部成员（含用户信息）
	GetMembers(roomID uint) ([]model.RoomMember, error)
}

// RoomServiceImpl RoomService 的默认实现
type RoomServiceImpl struct {
	roomRepo repository.RoomRepository
}

// NewRoomService 创建房间服务实例
func NewRoomService(roomRepo repository.RoomRepository) RoomService {
	return &RoomServiceImpl{roomRepo: roomRepo}
}

// CreateRoom 创建房间，创建者自动成为群主成员（role=owner）
func (s *RoomServiceImpl) CreateRoom(name, description, roomType, avatar string, ownerID uint) (*model.Room, error) {
	// 1. 校验房间名称非空，类型缺省为公开房间
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrRoomNameEmpty
	}
	if roomType == "" {
		roomType = "public"
	}

	// 2. 创建房间
	room := &model.Room{
		Name:        name,
		Description: strings.TrimSpace(description),
		Type:        roomType,
		Avatar:      strings.TrimSpace(avatar),
		CreatedBy:   ownerID,
	}
	if err := s.roomRepo.CreateRoom(room); err != nil {
		return nil, err
	}

	// 3. 创建者自动成为群主成员
	member := &model.RoomMember{
		RoomID: room.ID,
		UserID: ownerID,
		Role:   "owner",
	}
	if err := s.roomRepo.AddMember(member); err != nil {
		return nil, err
	}

	// 4. 填充成员数量（创建者本人）
	room.MemberCount = 1
	return room, nil
}

// JoinRoom 用户加入房间（已是成员时返回 ErrAlreadyMember）
func (s *RoomServiceImpl) JoinRoom(roomID, userID uint) error {
	// 1. 校验房间存在
	if _, err := s.roomRepo.GetRoomByID(roomID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrRoomNotFound
		}
		return err
	}

	// 2. 避免重复加入
	isMember, err := s.roomRepo.IsMember(roomID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyMember
	}

	// 3. 写入成员记录，普通成员角色
	member := &model.RoomMember{
		RoomID: roomID,
		UserID: userID,
		Role:   "member",
	}
	return s.roomRepo.AddMember(member)
}

// LeaveRoom 用户退出房间（非成员或群主退出时返回错误）
func (s *RoomServiceImpl) LeaveRoom(roomID, userID uint) error {
	// 1. 校验用户确实是房间成员
	role, err := s.roomRepo.GetMemberRole(roomID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotMember
		}
		return err
	}

	// 2. 群主不能退出（防止房间无人管理）
	if role == "owner" {
		return ErrOwnerCannotLeave
	}

	// 3. 移除成员记录
	return s.roomRepo.RemoveMember(roomID, userID)
}

// GetRoom 查询房间基本信息（附带成员数量）
func (s *RoomServiceImpl) GetRoom(roomID uint) (*model.Room, error) {
	room, err := s.roomRepo.GetRoomByID(roomID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}

	// 查询并填充成员数量
	counts, err := s.roomRepo.GetMemberCounts([]uint{roomID})
	if err != nil {
		return nil, err
	}
	room.MemberCount = counts[roomID]
	return room, nil
}

// GetUserRooms 查询用户加入的所有房间（附带各房间成员数量）
func (s *RoomServiceImpl) GetUserRooms(userID uint) ([]model.Room, error) {
	rooms, err := s.roomRepo.GetRoomsByUser(userID)
	if err != nil {
		return nil, err
	}
	if len(rooms) == 0 {
		return rooms, nil
	}

	// 批量统计成员数量并填充（GROUP BY 一次查询，避免 N+1）
	ids := make([]uint, 0, len(rooms))
	for _, room := range rooms {
		ids = append(ids, room.ID)
	}
	counts, err := s.roomRepo.GetMemberCounts(ids)
	if err != nil {
		return nil, err
	}
	for i := range rooms {
		rooms[i].MemberCount = counts[rooms[i].ID]
	}
	return rooms, nil
}

// GetMembers 查询房间的全部成员（含用户信息）
func (s *RoomServiceImpl) GetMembers(roomID uint) ([]model.RoomMember, error) {
	// 校验房间存在
	if _, err := s.roomRepo.GetRoomByID(roomID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	return s.roomRepo.GetMembers(roomID)
}
