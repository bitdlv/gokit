package idgen

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// QuerySerialMaxSeqFunc 根据业务 code 从 DB 查询当前最大序号（末段整数）。
type QuerySerialMaxSeqFunc func(ctx context.Context, code string) (int64, error)

// SerialGenerator 生成 "{code}.{NNNNNN}" 形式的流水号（如物料编号 A001-0001.000001）。
//
// 支持按业务前缀路由到不同 QueryMax 回调（例如"模块"与"普通物料"查询逻辑不同）。
type SerialGenerator struct {
	rdb       *redis.Client
	keyPrefix string
	seqWidth  int
	// routers 按 code 前缀选择不同的 QueryMax 实现；未匹配则使用 defaultQuery
	routers      []serialRouter
	defaultQuery QuerySerialMaxSeqFunc
	lua          *redis.Script
}

type serialRouter struct {
	prefix string
	query  QuerySerialMaxSeqFunc
}

// NewSerialGenerator 创建流水号生成器。
//   - redisKeyPrefix 例如 "bom:serial:"（可空，默认 "serial:"）
//   - seqWidth<=0 时默认 6
//   - defaultQuery 必填
func NewSerialGenerator(rdb *redis.Client, redisKeyPrefix string, seqWidth int, defaultQuery QuerySerialMaxSeqFunc) *SerialGenerator {
	if seqWidth <= 0 {
		seqWidth = 6
	}
	if redisKeyPrefix == "" {
		redisKeyPrefix = "serial:"
	}
	lua := redis.NewScript(`
local val = redis.call("GET", KEYS[1])
if not val then
    return 0
end
return redis.call("INCR", KEYS[1])
`)
	return &SerialGenerator{
		rdb:          rdb,
		keyPrefix:    redisKeyPrefix,
		seqWidth:     seqWidth,
		defaultQuery: defaultQuery,
		lua:          lua,
	}
}

// RegisterPrefixQuery 为指定 code 前缀注册专用查询回调（如 ModuleCodePrefix）。
// 匹配按注册顺序生效。
func (g *SerialGenerator) RegisterPrefixQuery(prefix string, query QuerySerialMaxSeqFunc) *SerialGenerator {
	g.routers = append(g.routers, serialRouter{prefix: prefix, query: query})
	return g
}

// NextCode 生成 "{code}.{NNNNNN}"。
func (g *SerialGenerator) NextCode(ctx context.Context, code string) (string, error) {
	redisKey := g.keyPrefix + code
	res, err := g.lua.Run(ctx, g.rdb, []string{redisKey}).Result()
	if err != nil {
		return "", fmt.Errorf("serial lua run failed: %w", err)
	}
	seq := res.(int64)
	if seq != 0 {
		return fmt.Sprintf("%s.%0*d", code, g.seqWidth, seq), nil
	}

	query := g.selectQuery(code)
	if query == nil {
		return "", fmt.Errorf("serial: no query registered for code=%s", code)
	}
	maxSeq, err := query(ctx, code)
	if err != nil {
		return "", err
	}
	if err := g.rdb.Set(ctx, redisKey, maxSeq, 0).Err(); err != nil {
		return "", fmt.Errorf("serial redis set failed: %w", err)
	}
	res, err = g.lua.Run(ctx, g.rdb, []string{redisKey}).Result()
	if err != nil {
		return "", fmt.Errorf("serial lua run after init failed: %w", err)
	}
	seq = res.(int64)
	return fmt.Sprintf("%s.%0*d", code, g.seqWidth, seq), nil
}

func (g *SerialGenerator) selectQuery(code string) QuerySerialMaxSeqFunc {
	for _, r := range g.routers {
		if len(code) >= len(r.prefix) && code[:len(r.prefix)] == r.prefix {
			return r.query
		}
	}
	return g.defaultQuery
}
