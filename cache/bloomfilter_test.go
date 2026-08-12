package cache

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"testing"
)

func TestBloomFilter(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis 地址
		Password: "",               // 密码
		DB:       0,                // 使用默认 DB
	})
	// 初始化布隆过滤器
	bloomFilter := NewBloomFilter(rdb, "bloomfilter:key", 32)

	// 添加元素到布隆过滤器
	bloomFilter.Add(context.Background(), "this is fliter value") //如用户ID

	// 检查元素是否可能存在
	exists, err := bloomFilter.Exists(context.Background(), fmt.Sprintf("%d", "this is fliter value"))
	if err != nil {
		log.Fatalf("Failed to check bloom filter: %v", err)
	}
	if !exists {
		fmt.Println("Data does not exist in bloom filter")
		return
	}
}
