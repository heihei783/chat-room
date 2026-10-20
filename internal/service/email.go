package service

import (
	"chat-app/internal/repository"
	"chat-app/pkg/email"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 验证码用途常量：区分注册、重置密码等场景，Redis键互不干扰
const (
	// PurposeRegister 验证码用途：注册
	PurposeRegister = "register"

	// PurposeResetPassword 验证码用途：重置密码
	PurposeResetPassword = "reset_password"
)

// 邮件服务常量
const (
	// verificationCodeTTL 验证码有效期（10分钟），存入Redis后自动过期
	verificationCodeTTL = 10 * time.Minute

	// maxVerifyAttempts 验证码最大尝试次数，超过后作废
	maxVerifyAttempts = 5
)

// 邮件服务错误定义：供上层(handler)区分错误类型并映射HTTP状态码
var (
	// ErrCodeNotFound 验证码不存在或已失效（未获取/已过期/已被使用）
	ErrCodeNotFound = errors.New("验证码不存在或已失效，请重新获取")
	// ErrCodeMismatch 验证码错误
	ErrCodeMismatch = errors.New("验证码错误")
)

// EmailService 邮件业务逻辑层接口，封装验证码的生成、发送与校验
type EmailService interface {
	// SendVerificationCode 发送注册验证码（Redis键按用途隔离，覆盖该邮箱旧码）
	SendVerificationCode(email string) error

	// SendResetCode 发送密码重置验证码（Redis键按用途隔离，覆盖该邮箱旧码）
	SendResetCode(email string) error

	// VerifyCode 校验指定用途的验证码，校验成功后删除该码防止重复使用
	VerifyCode(email, code, purpose string) error
}

// EmailServiceImpl EmailService 的默认实现
type EmailServiceImpl struct {
	codeRepo repository.EmailCodeRepository
	sender   *email.Sender // SMTP邮件发送器
}

// NewEmailService 创建邮件服务实例
func NewEmailService(codeRepo repository.EmailCodeRepository, sender *email.Sender) EmailService {
	return &EmailServiceImpl{
		codeRepo: codeRepo,
		sender:   sender,
	}
}

// SendVerificationCode 发送注册验证码（Redis键按用途隔离，覆盖该邮箱旧码）
func (s *EmailServiceImpl) SendVerificationCode(email string) error {
	return s.sendCode(email, PurposeRegister, "【聊天应用】邮箱验证码")
}

// SendResetCode 发送密码重置验证码（Redis键按用途隔离，覆盖该邮箱旧码）
func (s *EmailServiceImpl) SendResetCode(email string) error {
	return s.sendCode(email, PurposeResetPassword, "【聊天应用】密码重置验证码")
}

// sendCode 统一的发码流程：删除旧码→生成新码写入Redis（TTL自动过期）→发送邮件
func (s *EmailServiceImpl) sendCode(email, purpose, subject string) error {
	// 1. 规范化邮箱（转小写、去空格），并删除该邮箱该用途的旧验证码
	email = strings.ToLower(strings.TrimSpace(email))
	if err := s.codeRepo.Delete(email, purpose); err != nil {
		return err
	}

	// 2. 生成6位随机数字验证码并写入Redis（TTL到期自动过期）
	code, err := generateCode()
	if err != nil {
		return err
	}
	if err := s.codeRepo.Set(email, code, purpose, verificationCodeTTL); err != nil {
		return err
	}

	// 3. 发送验证码邮件（含有效期提示）
	body := fmt.Sprintf("您好，您的验证码是：%s，请在%d分钟内完成验证。如非本人操作，请忽略本邮件。",
		code, int(verificationCodeTTL.Minutes()))
	return s.sender.Send(email, subject, body)
}

// VerifyCode 校验指定用途的验证码，校验成功后删除该码防止重复使用
func (s *EmailServiceImpl) VerifyCode(email, code, purpose string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)

	// 1. 从Redis读取验证码及其尝试次数（键不存在即视为未获取/已过期）
	stored, attempts, err := s.codeRepo.Get(email, purpose)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCodeNotFound
		}
		return err
	}

	// 2. 比对验证码；失败时累计尝试次数，超过上限直接作废该验证码
	if stored != code {
		if attempts+1 >= maxVerifyAttempts {
			_ = s.codeRepo.Delete(email, purpose)
		}
		_ = s.codeRepo.IncrementAttempts(email, purpose)
		return ErrCodeMismatch
	}

	// 3. 校验成功，删除验证码，防止被重复使用
	return s.codeRepo.Delete(email, purpose)
}

// generateCode 生成6位数字验证码（crypto/rand 保证随机性，避免被预测）
func generateCode() (string, error) {
	const digits = "0123456789"
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成验证码失败: %w", err)
	}
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b), nil
}
