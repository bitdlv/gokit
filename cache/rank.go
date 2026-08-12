package cache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
)

type Rank struct {
	client *redis.Client
}

type Page struct {
	Total   int64     // 总元素数量
	Data    []redis.Z // 本页数据（带 score 的）
	Page    int64     // 当前页码
	Size    int64     // 每页大小
	HasNext bool      // 是否还有下一页
}

func NewRank(client *redis.Client) *Rank {
	return &Rank{client: client}
}

func (r *Rank) AddOrUpdateScore(ctx context.Context, column, key string, value float64) error {
	return r.client.ZAdd(ctx, column, &redis.Z{Score: value, Member: key}).Err()
}

// GetRankPage 分页查询
func (r *Rank) GetRankPage(ctx context.Context, column string, page, pageSize int64) (*Page, error) {
	if page < 1 {
		page = 1
	}
	start := (page - 1) * pageSize
	stop := start + pageSize - 1

	// 使用 pipeline 提高效率
	pipe := r.client.Pipeline()
	rangeCmd := pipe.ZRevRangeWithScores(ctx, column, start, stop)
	countCmd := pipe.ZCard(ctx, column)
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("cmd exec err: %v", err)
		return nil, err
	}

	total := countCmd.Val()
	data := rangeCmd.Val()
	return &Page{
		Total:   total,
		Data:    data,
		Page:    page,
		Size:    pageSize,
		HasNext: (page * pageSize) < total,
	}, nil
}

// GetUserRank 获取元素排名
func (r *Rank) GetUserRank(ctx context.Context, column string, key string) (int64, error) {
	return r.client.ZRevRank(ctx, column, key).Result()
}
