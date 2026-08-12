package cache

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"log"
	"time"
)

type DistributedLock struct {
	client *redis.Client
}

func NewDistributedLock(client *redis.Client) *DistributedLock {
	return &DistributedLock{client: client}
}

// AcquireLock 获取分布式锁
func (dl *DistributedLock) AcquireLock(ctx context.Context, lockKey string, timeout time.Duration) (string, bool, error) {
	lockValue := uuid.New().String()
	result, err := dl.client.SetNX(ctx, lockKey, lockValue, timeout).Result()
	if err != nil {
		return "", false, fmt.Errorf("failed to acquire lock: %v", err)
	}
	return lockValue, result, nil
}

// RenewLock 续期分布式锁
func (dl *DistributedLock) RenewLock(ctx context.Context, lockKey string, lockValue string, timeout time.Duration) {
	ticker := time.NewTicker(timeout / 2) // 每隔一半的过期时间续期一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 使用 Lua 脚本续期锁
			script := `
			if redis.call("get", KEYS[1]) == ARGV[1] then
				return redis.call("expire", KEYS[1], ARGV[2])
			else
				return 0
			end
			`
			_, err := dl.client.Eval(ctx, script, []string{lockKey}, lockValue, int(timeout.Seconds())).Result()
			if err != nil {
				log.Printf("Failed to renew lock: %v", err)
				return
			}
		case <-ctx.Done():
			// 上下文取消时停止续期
			return
		}
	}
}

// ReleaseLock 释放分布式锁
func (dl *DistributedLock) ReleaseLock(ctx context.Context, lockKey string, lockValue string) error {
	// 使用 Lua 脚本确保只有锁的值匹配时才释放锁
	script := `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
	`
	_, err := dl.client.Eval(ctx, script, []string{lockKey}, lockValue).Result()
	return err
}
