package cache

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func TestCache(t *testing.T) {
	// 初始化 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis 地址
		Password: "",               // 密码
		DB:       0,                // 使用默认 DB
	})
	// 初始化缓存和分布式锁
	cache := NewCache(rdb)
	distLock := NewDistributedLock(rdb)
	// 初始化缓存生成器
	cacheGen := NewCacheGenerator(cache, distLock)
	// 生成缓存数据（带分布式锁）
	data, err := cacheGen.GenerateWithLock(
		context.Background(),
		fmt.Sprintf("cache:key:%s", ""),
		fmt.Sprintf("locker:key:%s", ""),
		time.Second,    // 锁的过期时间
		10*time.Minute, // 缓存过期时间
		func() (string, error) {
			//这里实现需要缓存数据的逻辑
			return "this is cache data", nil
		}, // 业务逻辑函数
	)
	if err != nil {
		log.Fatalf("Failed to generate cache data: %v", err)
	}
	fmt.Printf("return data : %s\n", data)
}
