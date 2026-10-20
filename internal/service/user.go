package service

import (
	"chat-app/internal/model"
	"chat-app/internal/repository"
	"chat-app/pkg/auth"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// 用户服务错误定义：供上层(handler)区分错误类型并映射HTTP状态码
var (
	// ErrEmailExists 邮箱已被注册
	ErrEmailExists = errors.New("邮箱已被注册")
	// ErrUserNotFound 用户不存在
	ErrUserNotFound = errors.New("用户不存在")
	// ErrInvalidPassword 邮箱或密码错误（统一提示，避免泄露邮箱是否已注册）
	ErrInvalidPassword = errors.New("邮箱或密码错误")
	// ErrWeakPassword 密码强度不足（少于6位）
	ErrWeakPassword = errors.New("密码长度不能少于6位")
)

// LoginResult 登录成功结果，包含JWT令牌与用户信息
type LoginResult struct {
	Token string      // JWT访问令牌
	User  *model.User // 登录用户信息
}

// UserService 用户业务逻辑层接口，封装注册、登录、资料管理等能力
type UserService interface {
	// Register 注册新用户（依据邮箱+验证码），返回创建后的用户
	Register(username, email, password, nickname, code string) (*model.User, error)

	// Login 用户登录（依据邮箱+密码），校验通过后签发JWT并记录会话
	Login(email, password, ip, userAgent string) (*LoginResult, error)

	// GetProfile 获取用户资料
	GetProfile(userID uint) (*model.User, error)

	// UpdateProfile 更新用户资料（昵称、头像）
	UpdateProfile(userID uint, nickname, avatar string) error

	// ResetPassword 重置密码（忘记密码场景）：校验邮箱验证码后更新为新密码
	ResetPassword(email, code, newPassword string) error
}

// UserServiceImpl UserService 的默认实现
type UserServiceImpl struct {
	userRepo     repository.UserRepository
	emailService EmailService // 邮件服务，用于注册时校验邮箱验证码
	tokenTTL     time.Duration // JWT令牌有效期，来自配置
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, emailService EmailService, tokenTTL time.Duration) UserService {
	return &UserServiceImpl{
		userRepo:     userRepo,
		emailService: emailService,
		tokenTTL:     tokenTTL,
	}
}

// Register 注册流程：校验密码强度→检查邮箱是否已注册→校验邮箱验证码→bcrypt加密密码→写入数据库
func (s *UserServiceImpl) Register(username, email, password, nickname, code string) (*model.User, error) {
	// 1. 校验密码强度（至少6位）
	if len(password) < 6 {
		return nil, ErrWeakPassword
	}

	// 2. 统一邮箱为小写并去除首尾空格，避免大小写差异导致重复注册
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)

	// 3. 检查邮箱是否已被注册
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, ErrEmailExists
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// 4. 校验该邮箱的注册验证码（由邮件服务校验，成功后验证码作废）
	if err := s.emailService.VerifyCode(email, code, PurposeRegister); err != nil {
		return nil, err
	}

	// 5. 使用bcrypt加密密码，数据库中不保存明文
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 6. 组装用户并写入数据库，昵称为空时默认取用户名
	user := &model.User{
		Username: username,
		Email:    email,
		Password: string(hashed),
		Nickname: strings.TrimSpace(nickname),
		Status:   "offline",
	}
	if user.Nickname == "" {
		user.Nickname = user.Username
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

// Login 登录流程：按邮箱查询用户→校验密码→签发JWT→记录会话→更新在线状态
func (s *UserServiceImpl) Login(email, password, ip, userAgent string) (*LoginResult, error) {
	// 1. 统一邮箱格式后按邮箱查询用户
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// 用户不存在时返回与密码错误相同的提示，避免泄露邮箱是否已注册
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidPassword
		}
		return nil, err
	}

	// 2. 校验密码是否与密文匹配
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidPassword
	}

	// 3. 签发JWT令牌（当前角色固定为普通用户）
	token, err := auth.GenerateToken(user.ID, user.Username, "user", s.tokenTTL)
	if err != nil {
		return nil, err
	}

	// 4. 记录登录会话（令牌、客户端IP、UserAgent、过期时间），用于后续登出与令牌管理
	now := time.Now()
	session := &model.UserSession{
		UserID:    user.ID,
		Token:     token,
		IP:        ip,
		UserAgent: userAgent,
		ExpiresAt: now.Add(s.tokenTTL),
	}
	if err := s.userRepo.CreateSession(session); err != nil {
		return nil, err
	}

	// 5. 更新在线状态与最后活跃时间（非关键操作，失败不影响登录结果）
	_ = s.userRepo.UpdateStatus(user.ID, "online")
	_ = s.userRepo.UpdateLastSeen(user.ID, now)

	return &LoginResult{Token: token, User: user}, nil
}

// GetProfile 获取用户资料，用户不存在时返回 ErrUserNotFound
func (s *UserServiceImpl) GetProfile(userID uint) (*model.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// UpdateProfile 更新用户昵称与头像，仅更新非空字段
func (s *UserServiceImpl) UpdateProfile(userID uint, nickname, avatar string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 仅当传入字段非空时才进行更新
	if nickname != "" {
		user.Nickname = nickname
	}
	if avatar != "" {
		user.Avatar = avatar
	}
	return s.userRepo.Update(user)
}

// ResetPassword 重置密码流程：校验密码强度→按邮箱查用户→校验重置验证码→bcrypt加密→更新密码
func (s *UserServiceImpl) ResetPassword(email, code, newPassword string) error {
	// 1. 校验新密码强度（至少6位）
	if len(newPassword) < 6 {
		return ErrWeakPassword
	}

	// 2. 统一邮箱为小写并去除首尾空格
	email = strings.ToLower(strings.TrimSpace(email))

	// 3. 按邮箱查询用户（不存在说明该邮箱未注册）
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 4. 校验密码重置验证码（校验成功后验证码作废）
	if err := s.emailService.VerifyCode(email, code, PurposeResetPassword); err != nil {
		return err
	}

	// 5. bcrypt加密新密码并更新（仅更新密码字段）
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(user.ID, string(hashed))
}
