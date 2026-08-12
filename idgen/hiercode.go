package idgen

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// QueryHierMaxSeqFunc 三层编码生成器每层的最大序号查询回调。
//
//	Level 分类：key 传 level(int) 的字符串形式
//	Level SPU ：key 传 categoryCode
//	Level SKU ：key 传 spuCode
type QueryHierMaxSeqFunc func(ctx context.Context, key string) (int64, error)

// HierCodeConfig 三层编码格式配置。
// 默认：分类 3 位、SPU 4 位、SKU 5 位（例如 A001 / A001-0001 / A001-0001-00001）。
type HierCodeConfig struct {
	CategoryWidth int
	SpuWidth      int
	SkuWidth      int
	// LevelPrefix 将 level(1-based) 转字母前缀。默认：1→A, 26→Z, 27→AA。
	LevelPrefix func(level int) string
}

// HierCodeGenerator 三级分层编码生成器：分类 → SPU → SKU。
type HierCodeGenerator struct {
	rdb                 *redis.Client
	cfg                 HierCodeConfig
	queryCategoryMaxSeq QueryHierMaxSeqFunc
	querySpuMaxSeq      QueryHierMaxSeqFunc
	querySkuMaxSeq      QueryHierMaxSeqFunc
	lua                 *redis.Script
}

// NewHierCodeGenerator 创建三层分层编码生成器。零值配置使用默认宽度和字母前缀映射。
func NewHierCodeGenerator(
	rdb *redis.Client,
	cfg HierCodeConfig,
	queryCategoryMaxSeq, querySpuMaxSeq, querySkuMaxSeq QueryHierMaxSeqFunc,
) *HierCodeGenerator {
	if cfg.CategoryWidth <= 0 {
		cfg.CategoryWidth = 3
	}
	if cfg.SpuWidth <= 0 {
		cfg.SpuWidth = 4
	}
	if cfg.SkuWidth <= 0 {
		cfg.SkuWidth = 5
	}
	if cfg.LevelPrefix == nil {
		cfg.LevelPrefix = DefaultLevelPrefix
	}
	lua := redis.NewScript(`
local val = redis.call("GET", KEYS[1])
if not val then
    return 0
end
return redis.call("INCR", KEYS[1])
`)
	return &HierCodeGenerator{
		rdb:                 rdb,
		cfg:                 cfg,
		queryCategoryMaxSeq: queryCategoryMaxSeq,
		querySpuMaxSeq:      querySpuMaxSeq,
		querySkuMaxSeq:      querySkuMaxSeq,
		lua:                 lua,
	}
}

// DefaultLevelPrefix 层级 → 字母前缀映射（1→A, 26→Z, 27→AA, 28→AB, ...）。
func DefaultLevelPrefix(level int) string {
	if level <= 0 {
		level = 1
	}
	level--
	var prefix string
	for {
		rem := level % 26
		prefix = string(rune('A'+rem)) + prefix
		level = level/26 - 1
		if level < 0 {
			break
		}
	}
	return prefix
}

// NextCategoryCode 返回 (code, seqNum, error)，例如 ("A001", 1, nil)。
func (g *HierCodeGenerator) NextCategoryCode(ctx context.Context, level int) (string, int64, error) {
	redisKey := fmt.Sprintf("hier:category:seq:level:%d", level)
	seq, err := g.incrOrInit(ctx, redisKey, func(ctx context.Context) (int64, error) {
		return g.queryCategoryMaxSeq(ctx, fmt.Sprintf("%d", level))
	})
	if err != nil {
		return "", 0, err
	}
	prefix := g.cfg.LevelPrefix(level)
	code := fmt.Sprintf("%s%0*d", prefix, g.cfg.CategoryWidth, seq)
	return code, seq, nil
}

// NextSpuCode 生成 {categoryCode}-{seq}。
func (g *HierCodeGenerator) NextSpuCode(ctx context.Context, categoryCode string) (string, error) {
	redisKey := fmt.Sprintf("hier:spu:seq:category:%s", categoryCode)
	seq, err := g.incrOrInit(ctx, redisKey, func(ctx context.Context) (int64, error) {
		return g.querySpuMaxSeq(ctx, categoryCode)
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%0*d", categoryCode, g.cfg.SpuWidth, seq), nil
}

// NextSkuCode 生成 {spuCode}-{seq}。
func (g *HierCodeGenerator) NextSkuCode(ctx context.Context, spuCode string) (string, error) {
	redisKey := fmt.Sprintf("hier:sku:seq:spu:%s", spuCode)
	seq, err := g.incrOrInit(ctx, redisKey, func(ctx context.Context) (int64, error) {
		return g.querySkuMaxSeq(ctx, spuCode)
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%0*d", spuCode, g.cfg.SkuWidth, seq), nil
}

func (g *HierCodeGenerator) incr(ctx context.Context, key string) (int64, error) {
	res, err := g.lua.Run(ctx, g.rdb, []string{key}).Result()
	if err != nil {
		return 0, fmt.Errorf("hiercode lua run failed key=%s: %w", key, err)
	}
	return res.(int64), nil
}

func (g *HierCodeGenerator) incrOrInit(ctx context.Context, key string, query func(ctx context.Context) (int64, error)) (int64, error) {
	seq, err := g.incr(ctx, key)
	if err != nil {
		return 0, err
	}
	if seq != 0 {
		return seq, nil
	}
	maxSeq, err := query(ctx)
	if err != nil {
		return 0, err
	}
	if err := g.rdb.Set(ctx, key, maxSeq, 0).Err(); err != nil {
		return 0, fmt.Errorf("hiercode redis set failed: %w", err)
	}
	return g.incr(ctx, key)
}
