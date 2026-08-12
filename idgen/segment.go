package idgen

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	defaultSegmentStep int64   = 1000
	prefetchThreshold  float64 = 0.2 // 剩余低于 20% 触发异步预取
)

// setIfGreaterScript 原子地将 Redis key 设置为 max(current, argv[1])。
var setIfGreaterScript = redis.NewScript(`
local cur = tonumber(redis.call('GET', KEYS[1])) or 0
local new = tonumber(ARGV[1])
if new > cur then
    redis.call('SET', KEYS[1], tostring(new))
    return new
end
return cur
`)

type idSegment struct {
	start int64
	end   int64 // exclusive
}

// SegmentGenerator 分段式 ID 预生成器：
//   - 基于 Redis INCRBY 原子申请段，集群唯一
//   - 首次使用时对齐到 max(db_id)（若配置了 db 则查询 SELECT MAX(id) FROM <name>）
//   - 本地缓冲当前段 + 异步预取下一段，减少 Redis 调用
//   - ID 范围 [1, MaxInt64)
type SegmentGenerator struct {
	client    *redis.Client
	db        *gorm.DB // 可为 nil：跳过 max(id) 校验
	tableName string
	key       string
	step      int64

	initOnce sync.Once

	mu          sync.Mutex
	curr        idSegment
	curPos      int64
	next        idSegment
	nextReady   atomic.Bool
	prefetching atomic.Bool
}

// NewSegmentGenerator 直接构造单个命名生成器；一般通过 Manager 管理。
func NewSegmentGenerator(client *redis.Client, db *gorm.DB, name string, step int64) *SegmentGenerator {
	if step <= 0 {
		step = defaultSegmentStep
	}
	return &SegmentGenerator{
		client:    client,
		db:        db,
		tableName: name,
		key:       fmt.Sprintf("idgen:%s", name),
		step:      step,
	}
}

func (g *SegmentGenerator) ensureInit(ctx context.Context) error {
	var initErr error
	g.initOnce.Do(func() {
		if g.db == nil {
			return
		}
		var maxID int64
		row := g.db.WithContext(ctx).
			Raw(fmt.Sprintf("SELECT COALESCE(MAX(id), 0) FROM `%s`", g.tableName)).
			Row()
		if err := row.Scan(&maxID); err != nil {
			initErr = fmt.Errorf("idgen[%s]: query max id failed: %w", g.tableName, err)
			return
		}
		if maxID <= 0 {
			return
		}
		if err := setIfGreaterScript.Run(ctx, g.client, []string{g.key}, maxID).Err(); err != nil && err != redis.Nil {
			initErr = fmt.Errorf("idgen[%s]: init redis counter failed: %w", g.tableName, err)
		}
	})
	return initErr
}

// Next 获取下一个 ID，线程安全。
func (g *SegmentGenerator) Next(ctx context.Context) (int64, error) {
	if err := g.ensureInit(ctx); err != nil {
		logx.Errorf("idgen[%s]: initialization error: %v", g.tableName, err)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	for {
		if g.curPos < g.curr.end {
			id := g.curPos
			g.curPos++
			used := float64(g.curPos-g.curr.start) / float64(g.curr.end-g.curr.start)
			if used >= (1-prefetchThreshold) && !g.nextReady.Load() {
				g.triggerPrefetch(ctx)
			}
			return id, nil
		}
		if g.nextReady.Load() {
			g.curr = g.next
			g.curPos = g.curr.start
			g.next = idSegment{}
			g.nextReady.Store(false)
			continue
		}
		seg, err := g.fetchSegment(ctx)
		if err != nil {
			return 0, err
		}
		g.curr = seg
		g.curPos = seg.start
	}
}

// NextN 批量获取 n 个 ID，返回单调递增切片；跨段时不保证严格连续。
func (g *SegmentGenerator) NextN(ctx context.Context, n int) ([]int64, error) {
	if n <= 0 {
		return nil, nil
	}
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		id, err := g.Next(ctx)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (g *SegmentGenerator) fetchSegment(ctx context.Context) (idSegment, error) {
	end, err := g.client.IncrBy(ctx, g.key, g.step).Result()
	if err != nil {
		return idSegment{}, fmt.Errorf("idgen[%s]: fetch segment failed: %w", g.key, err)
	}
	start := end - g.step + 1
	if start < 1 {
		start = 1
	}
	return idSegment{start: start, end: end + 1}, nil
}

func (g *SegmentGenerator) triggerPrefetch(ctx context.Context) {
	if !g.prefetching.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer g.prefetching.Store(false)
		end, err := g.client.IncrBy(context.Background(), g.key, g.step).Result()
		if err != nil {
			return
		}
		start := end - g.step + 1
		if start < 1 {
			start = 1
		}
		seg := idSegment{start: start, end: end + 1}
		g.mu.Lock()
		if !g.nextReady.Load() {
			g.next = seg
			g.nextReady.Store(true)
		}
		g.mu.Unlock()
	}()
	_ = ctx
}

// SegmentManager 按名称管理多个 SegmentGenerator，懒初始化。
type SegmentManager struct {
	mu     sync.RWMutex
	gens   map[string]*SegmentGenerator
	client *redis.Client
	db     *gorm.DB
	step   int64
}

// NewSegmentManager 创建 Manager。step<=0 时使用默认 1000。
func NewSegmentManager(client *redis.Client, db *gorm.DB, step int64) *SegmentManager {
	if step <= 0 {
		step = defaultSegmentStep
	}
	return &SegmentManager{
		gens:   make(map[string]*SegmentGenerator),
		client: client,
		db:     db,
		step:   step,
	}
}

// Get 返回指定名称的生成器；不存在则创建。推荐用表名。
func (m *SegmentManager) Get(name string) *SegmentGenerator {
	m.mu.RLock()
	if g, ok := m.gens[name]; ok {
		m.mu.RUnlock()
		return g
	}
	m.mu.RUnlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	if g, ok := m.gens[name]; ok {
		return g
	}
	g := NewSegmentGenerator(m.client, m.db, name, m.step)
	m.gens[name] = g
	return g
}
