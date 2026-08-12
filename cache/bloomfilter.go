package cache

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)

type BloomFilter struct {
	client    *redis.Client
	shardBits int
	bitmapkey string
}

func NewBloomFilter(client *redis.Client, bitmapkey string, shardBits int) *BloomFilter {
	return &BloomFilter{client: client, shardBits: shardBits, bitmapkey: bitmapkey}
}

// 哈希函数
func hash(element string) int64 {
	var hash int64
	for _, char := range element {
		hash = hash*31 + int64(char)
	}
	return hash % (1 << 20)
}

// 获取分片 Key 和索引
func (bf *BloomFilter) getShardAndIndex(element string) (string, int64) {
	hashValue := hash(element)
	// 高 n 位作为分片 Key
	shardKey := fmt.Sprintf("%s:%d", bf.bitmapkey, hashValue>>bf.shardBits)
	// 低 m 位作为索引
	index := hashValue & ((1 << bf.shardBits) - 1)
	return shardKey, index
}

// 添加元素到布隆过滤器
func (bf *BloomFilter) Add(ctx context.Context, element string) error {
	shardKey, index := bf.getShardAndIndex(element)
	return bf.client.SetBit(ctx, shardKey, index, 1).Err()
}

// 检查元素是否可能存在
func (bf *BloomFilter) Exists(ctx context.Context, element string) (bool, error) {
	shardKey, index := bf.getShardAndIndex(element)
	result, err := bf.client.GetBit(ctx, shardKey, index).Result()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
