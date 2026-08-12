package redislocker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// 默认锁超时时间
	defaultLockTimeout = 30 * time.Second
	// 默认重试间隔
	defaultRetryInterval = 100 * time.Millisecond
	// 默认看门狗续期间隔
	defaultWatchdogInterval = 10 * time.Second
)

// Lock Redis分布式锁
type Lock struct {
	client     *redis.Client
	key        string        // 锁的key
	value      string        // 锁的唯一标识值
	expiration time.Duration // 锁的过期时间
	watchdog   chan struct{} // 看门狗信号通道
}

// NewLock 创建一个新的分布式锁
func NewLock(client *redis.Client, key string) *Lock {
	return &Lock{
		client:     client,
		key:        fmt.Sprintf("lock:%s", key),
		expiration: defaultLockTimeout,
		watchdog:   make(chan struct{}),
	}
}

// Lock 获取锁
func (l *Lock) Lock(ctx context.Context) error {
	// 生成随机值作为锁的唯一标识
	value, err := generateRandomValue()
	if err != nil {
		return fmt.Errorf("generate random value error: %w", err)
	}
	l.value = value

	// 尝试获取锁
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			success, err := l.tryLock(ctx)
			if err != nil {
				return err
			}
			if success {
				// 启动看门狗续期
				go l.startWatchdog(ctx)
				return nil
			}
			// 等待一段时间后重试
			time.Sleep(defaultRetryInterval)
		}
	}
}

// TryLock 尝试获取锁，如果获取不到立即返回
func (l *Lock) TryLock(ctx context.Context) (bool, error) {
	value, err := generateRandomValue()
	if err != nil {
		return false, fmt.Errorf("generate random value error: %w", err)
	}
	l.value = value

	success, err := l.tryLock(ctx)
	if err != nil {
		return false, err
	}
	if success {
		go l.startWatchdog(ctx)
	}
	return success, nil
}

// Unlock 释放锁
func (l *Lock) Unlock(ctx context.Context) error {
	// 使用Lua脚本确保原子性操作
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	// 停止看门狗
	close(l.watchdog)

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Result()
	if err != nil {
		return fmt.Errorf("unlock error: %w", err)
	}
	if result.(int64) == 0 {
		return fmt.Errorf("lock not held")
	}
	return nil
}

// 内部方法

// tryLock 尝试获取锁
func (l *Lock) tryLock(ctx context.Context) (bool, error) {
	success, err := l.client.SetNX(ctx, l.key, l.value, l.expiration).Result()
	if err != nil {
		return false, fmt.Errorf("set lock error: %w", err)
	}
	return success, nil
}

// startWatchdog 启动看门狗进行锁续期
func (l *Lock) startWatchdog(ctx context.Context) {
	ticker := time.NewTicker(defaultWatchdogInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.watchdog:
			return
		case <-ticker.C:
			l.refresh(ctx)
		}
	}
}

// refresh 刷新锁的过期时间
func (l *Lock) refresh(ctx context.Context) {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`
	expireMs := int64(l.expiration / time.Millisecond)
	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value, expireMs).Result()
	if err != nil {
		log.Printf("refresh lock error: %v", err)
		return
	}
	if result.(int64) == 0 {
		log.Printf("lock lost: %s", l.key)
		return
	}
}

// SetExpiration 设置锁的过期时间
func (l *Lock) SetExpiration(expiration time.Duration) {
	l.expiration = expiration
}

// generateRandomValue 生成随机值
func generateRandomValue() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
