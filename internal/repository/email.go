package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// EmailCodeRepository 邮箱验证码存储接口（基于Redis）
// 验证码为短时效临时数据，使用Redis存储，键带TTL自动过期，无需建表与手动清理；
// 键按用途（purpose）区分，注册、重置密码等场景的验证码互不干扰
type EmailCodeRepository interface {
	// Set 保存验证码并设置过期时间（同时重置该邮箱该用途的尝试次数）
	Set(email, code, purpose string, ttl time.Duration) error

	// Get 读取验证码及其当前尝试次数；验证码不存在或已过期时返回 ErrNotFound
	Get(email, purpose string) (code string, attempts int64, err error)

	// Delete 删除验证码及尝试次数（作废旧码或校验成功后调用）
	Delete(email, purpose string) error

	// IncrementAttempts 将验证码的尝试次数加1，并与验证码保持相同生命周期
	IncrementAttempts(email, purpose string) error
}

// EmailCodeRepositoryImpl 基于 Redis 的 EmailCodeRepository 实现
type EmailCodeRepositoryImpl struct {
	redis *redis.Client
}

// NewEmailCodeRepository 创建邮箱验证码仓储实例（需传入已初始化的Redis客户端）
func NewEmailCodeRepository(redis *redis.Client) EmailCodeRepository {
	return &EmailCodeRepositoryImpl{redis: redis}
}

// codeKey 验证码在Redis中的存储键（按用途区分）
func codeKey(email, purpose string) string {
	return fmt.Sprintf("email:code:%s:%s", purpose, email)
}

// attemptsKey 验证码尝试次数在Redis中的存储键（按用途区分）
func attemptsKey(email, purpose string) string {
	return fmt.Sprintf("email:attempts:%s:%s", purpose, email)
}

// Set 保存验证码并设置过期时间（同时重置该邮箱该用途的尝试次数）
func (r *EmailCodeRepositoryImpl) Set(email, code, purpose string, ttl time.Duration) error {
	ctx := context.Background()
	// 使用事务管道：写入验证码并重置尝试次数，保证原子性
	pipe := r.redis.TxPipeline()
	pipe.Set(ctx, codeKey(email, purpose), code, ttl)
	pipe.Del(ctx, attemptsKey(email, purpose))
	_, err := pipe.Exec(ctx)
	return err
}

// Get 读取验证码及其当前尝试次数；验证码不存在或已过期时返回 ErrNotFound
func (r *EmailCodeRepositoryImpl) Get(email, purpose string) (code string, attempts int64, err error) {
	ctx := context.Background()

	// 读取验证码（键不存在即视为未获取或已过期）
	code, err = r.redis.Get(ctx, codeKey(email, purpose)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", 0, ErrNotFound
		}
		return "", 0, err
	}

	// 读取尝试次数（可能尚未记录，忽略键不存在错误）
	attempts, err = r.redis.Get(ctx, attemptsKey(email, purpose)).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", 0, err
	}
	return code, attempts, nil
}

// Delete 删除验证码及尝试次数（作废旧码或校验成功后调用）
func (r *EmailCodeRepositoryImpl) Delete(email, purpose string) error {
	ctx := context.Background()
	pipe := r.redis.TxPipeline()
	pipe.Del(ctx, codeKey(email, purpose))
	pipe.Del(ctx, attemptsKey(email, purpose))
	_, err := pipe.Exec(ctx)
	return err
}

// IncrementAttempts 将验证码的尝试次数加1，并与验证码保持相同生命周期
func (r *EmailCodeRepositoryImpl) IncrementAttempts(email, purpose string) error {
	ctx := context.Background()
	key := attemptsKey(email, purpose)

	// 获取验证码剩余TTL，使次数键与验证码同时过期
	ttl, err := r.redis.TTL(ctx, codeKey(email, purpose)).Result()
	if err != nil {
		return err
	}

	pipe := r.redis.TxPipeline()
	pipe.Incr(ctx, key)
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	_, err = pipe.Exec(ctx)
	return err
}
