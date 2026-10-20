package repository

import (
	"chat-app/internal/model"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ErrNotFound 表示查询的记录不存在
var ErrNotFound = errors.New("记录不存在")

// UserRepository 用户数据访问层接口，抽象用户相关的数据库操作
type UserRepository interface {
	// Create 创建新用户，用于注册
	Create(user *model.User) error

	// FindByEmail 根据邮箱查询用户，用于登录与邮箱查重（邮箱字段唯一）
	FindByEmail(email string) (*model.User, error)

	// FindByID 根据用户ID查询用户
	FindByID(id uint) (*model.User, error)

	// FindByIDs 根据用户ID集合批量查询用户（用于在线用户列表等场景）
	FindByIDs(ids []uint) ([]model.User, error)

	// Update 更新用户信息（昵称、头像等）
	Update(user *model.User) error

	// UpdatePassword 更新用户密码（仅更新密码字段，避免全量保存覆盖其他并发修改）
	UpdatePassword(userID uint, password string) error

	// UpdateStatus 更新用户在线状态（online / offline / away）
	UpdateStatus(userID uint, status string) error

	// UpdateLastSeen 更新用户最后活跃时间
	UpdateLastSeen(userID uint, lastSeen time.Time) error

	// CreateSession 保存用户登录会话（JWT令牌等）
	CreateSession(session *model.UserSession) error

	// DeleteSession 根据令牌删除会话，用于登出
	DeleteSession(token string) error

	// DeleteExpiredSessions 清理所有已过期的会话记录
	DeleteExpiredSessions() error
}

// UserRepositoryImpl 基于 GORM 的 UserRepository 实现
type UserRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{db: db}
}

// Create 创建新用户，用于注册
func (r *UserRepositoryImpl) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// FindByEmail 根据邮箱查询用户，用于登录与邮箱查重（邮箱字段唯一）
func (r *UserRepositoryImpl) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 根据用户ID查询用户
func (r *UserRepositoryImpl) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByIDs 根据用户ID集合批量查询用户（用于在线用户列表等场景）
func (r *UserRepositoryImpl) FindByIDs(ids []uint) ([]model.User, error) {
	if len(ids) == 0 {
		return []model.User{}, nil
	}
	users := make([]model.User, 0)
	err := r.db.Where("id IN ?", ids).Find(&users).Error
	return users, err
}

// Update 更新用户信息（昵称、头像等）
func (r *UserRepositoryImpl) Update(user *model.User) error {
	return r.db.Save(user).Error
}

// UpdatePassword 更新用户密码（仅更新密码字段，避免全量保存覆盖其他并发修改）
func (r *UserRepositoryImpl) UpdatePassword(userID uint, password string) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("password", password).Error
}

// UpdateStatus 更新用户在线状态（online / offline / away）
func (r *UserRepositoryImpl) UpdateStatus(userID uint, status string) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("status", status).Error
}

// UpdateLastSeen 更新用户最后活跃时间
func (r *UserRepositoryImpl) UpdateLastSeen(userID uint, lastSeen time.Time) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("last_seen", lastSeen).Error
}

// CreateSession 保存用户登录会话（JWT令牌等）
func (r *UserRepositoryImpl) CreateSession(session *model.UserSession) error {
	return r.db.Create(session).Error
}

// DeleteSession 根据令牌删除会话，用于登出
func (r *UserRepositoryImpl) DeleteSession(token string) error {
	return r.db.Where("token = ?", token).Delete(&model.UserSession{}).Error
}

// DeleteExpiredSessions 清理所有已过期的会话记录
func (r *UserRepositoryImpl) DeleteExpiredSessions() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&model.UserSession{}).Error
}
