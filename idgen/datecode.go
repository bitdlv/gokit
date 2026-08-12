package idgen

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// QueryDateMaxSeqFunc 从 DB 查询指定前缀+日期下已有的最大序号。
// 常见实现：SELECT code ... WHERE code LIKE 'BID-20260416-%' ORDER BY code DESC LIMIT 1，
// 再解析出末段序号返回。
type QueryDateMaxSeqFunc func(ctx context.Context, prefix, dateStr string) (int64, error)

// DateCodeGenerator 生成 "{PREFIX}-YYYYMMDD-NNNNNN" 形式的日期序号编码。
//
// 语义：
//   - 每天独立计数，Redis key TTL 到当天 23:59:59
//   - 首次使用时通过 QueryDateMaxSeqFunc 从 DB 恢复最大序号
//
// 典型场景：报价单号 BID-20260416-000001、BOM 方案号 BOM-20260416-000001
type DateCodeGenerator struct {
	rdb      *redis.Client
	prefix   string
	seqWidth int // 序号宽度，默认 6
	keyBase  string
	query    QueryDateMaxSeqFunc
	lua      *redis.Script
}

// NewDateCodeGenerator 创建日期序号编码生成器。
// prefix 例如 "BID"；redisKeyBase 例如 "bid:code:"（可选，为空时用 "<lowerPrefix>:code:"）；
// seqWidth<=0 时默认 6。
func NewDateCodeGenerator(rdb *redis.Client, prefix, redisKeyBase string, seqWidth int, query QueryDateMaxSeqFunc) *DateCodeGenerator {
	if seqWidth <= 0 {
		seqWidth = 6
	}
	if redisKeyBase == "" {
		redisKeyBase = fmt.Sprintf("%s:code:", prefix)
	}
	lua := redis.NewScript(`
local val = redis.call("GET", KEYS[1])
if not val then
    return 0
end
return redis.call("INCR", KEYS[1])
`)
	return &DateCodeGenerator{
		rdb:      rdb,
		prefix:   prefix,
		seqWidth: seqWidth,
		keyBase:  redisKeyBase,
		query:    query,
		lua:      lua,
	}
}

// NextCode 生成当天的下一个编码。
func (g *DateCodeGenerator) NextCode(ctx context.Context) (string, error) {
	return g.NextCodeByDate(ctx, time.Now())
}

// NextCodeByDate 生成指定日期的下一个编码。
func (g *DateCodeGenerator) NextCodeByDate(ctx context.Context, date time.Time) (string, error) {
	dateStr := date.Format("20060102")
	redisKey := g.keyBase + dateStr

	res, err := g.lua.Run(ctx, g.rdb, []string{redisKey}).Result()
	if err != nil {
		return "", fmt.Errorf("datecode[%s] lua run failed: %w", g.prefix, err)
	}
	seq := res.(int64)
	if seq == 0 {
		var maxSeq int64
		if g.query != nil {
			maxSeq, err = g.query(ctx, g.prefix, dateStr)
			if err != nil {
				return "", fmt.Errorf("datecode[%s] query db failed: %w", g.prefix, err)
			}
		}
		if err := g.rdb.Set(ctx, redisKey, maxSeq, g.ttlUntilEndOfDay(date)).Err(); err != nil {
			logx.Errorf("datecode[%s] redis set failed: %v", g.prefix, err)
		}
		res, err = g.lua.Run(ctx, g.rdb, []string{redisKey}).Result()
		if err != nil {
			return "", fmt.Errorf("datecode[%s] lua run after init failed: %w", g.prefix, err)
		}
		seq = res.(int64)
	}

	return fmt.Sprintf("%s-%s-%0*d", g.prefix, dateStr, g.seqWidth, seq), nil
}

func (g *DateCodeGenerator) ttlUntilEndOfDay(date time.Time) time.Duration {
	loc := date.Location()
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, loc)
	ttl := time.Until(endOfDay)
	if ttl <= 0 {
		ttl = time.Second
	}
	return ttl
}
