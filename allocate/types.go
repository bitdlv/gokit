package allocate

import (
	"context"
	"time"

	"github.com/bitdlv/gokit/tree"
)

// Item 待分摊项（如费用、任务、资源等）
type Item struct {
	ID    string  // 唯一标识
	Name  string  // 名称
	Value int64   // 总量（如金额、数量、工时等）
	Ratio float64 // 比例因子（如税率、折扣率等），用于拆分净值，默认 0
}

// AllocDetail 单个 Item 在节点上的分摊结果
type AllocDetail struct {
	ItemID   string  `json:"item_id"`
	ItemName string  `json:"item_name"`
	Value    int64   `json:"value"`     // 分摊到的量
	ValueNet int64   `json:"value_net"` // 按 Ratio 拆分后的净值
	Ratio    float64 `json:"ratio"`     // 比例因子
}

// NodeDetail 节点完整明细
type NodeDetail struct {
	NodeID string `json:"node_id"`

	// 权重与基础值
	BaseWeight   float64 `json:"base_weight"`    // 计算权重（内部使用）
	BaseValue    int64   `json:"base_value"`     // 基础值（叶子：unit*qty，中间：sum(children)*qty）
	BaseValueNet int64   `json:"base_value_net"` // 净值基础值

	// 分摊明细
	AllocList  []*AllocDetail `json:"alloc_list"`  // 分摊明细列表
	AllocTotal int64          `json:"alloc_total"` // 分摊总额

	// 小计
	SubTotal    int64 `json:"sub_total"`     // 小计 = BaseValue + AllocTotal
	SubTotalNet int64 `json:"sub_total_net"` // 净值小计

	// 加成
	Extra    int64 `json:"extra"`     // 加成量
	Total    int64 `json:"total"`     // 总量
	TotalNet int64 `json:"total_net"` // 净值总量

	// 树形结构
	Children []*NodeDetail `json:"children,omitempty"`
}

// Result 计算结果
type Result struct {
	// 汇总
	TotalBase     int64 `json:"total_base"`      // 总基础值
	TotalAlloc    int64 `json:"total_alloc"`     // 总分摊额
	TotalExtra    int64 `json:"total_extra"`     // 总加成
	GrandTotal    int64 `json:"grand_total"`     // 总报价
	GrandTotalNet int64 `json:"grand_total_net"` // 净值总报价

	// 树形明细
	Roots []*NodeDetail `json:"roots"`

	// 平铺索引
	NodeIndex map[string]*NodeDetail `json:"-"` // 不序列化，内部使用
}

// Cache 缓存接口
type Cache interface {
	Get(ctx context.Context, key string) (*Result, error)
	Set(ctx context.Context, key string, result *Result, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// DataLoader 数据加载器接口
// 根据 bizID 和 bizVersion 加载对应的节点数据、分摊项、加成配置
//
// 实现示例：
//   - 从数据库加载：SELECT * FROM nodes WHERE biz_id = ? AND version = ?
//   - 从 Redis 加载：HGETALL allocate:data:{bizID}:v{bizVersion}
//   - 从内存加载：查共享 map
type DataLoader[T any] interface {
	// Load 加载指定业务ID和版本的数据
	// 返回: nodes, items, extraRate, extraRatio, error
	Load(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[T], []Item, float64, float64, error)
}

// DataLoaderFunc 函数式 DataLoader（简化实现）
type DataLoaderFunc[T any] func(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[T], []Item, float64, float64, error)

// Load 实现 DataLoader 接口
func (f DataLoaderFunc[T]) Load(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[T], []Item, float64, float64, error) {
	return f(ctx, bizID, bizVersion)
}

// ─────────────────────────────────────────────
// 配置选项
// ─────────────────────────────────────────────

// Option 计算器配置选项
type Option[T any] func(*Calculator[T])

// WithCacheKey 设置业务缓存键（必须）
// bizID: 业务ID（如项目ID、订单ID、测算ID）
// bizVersion: 业务版本号（数据变更时递增，由调用方维护）
//
// 缓存键格式: allocate:{bizID}:v{bizVersion}
// 示例: allocate:project:123:v5
//
// 注意：
//   - 相同 bizID+bizVersion 的 Calculator 实例共享缓存
//   - 数据变更后应递增 bizVersion，自动加载新数据
//   - 必须配合 WithDataLoader 使用，Calculate() 时自动加载数据
func WithCacheKey[T any](bizID string, bizVersion int64) Option[T] {
	return func(c *Calculator[T]) {
		c.bizID = bizID
		c.bizVersion = bizVersion
	}
}

// WithDataLoader 设置数据加载器（必须）
// Calculate() 时自动调用 loader.Load(bizID, bizVersion) 加载数据
//
// 示例：
//
//	allocate.WithDataLoader(allocate.DataLoaderFunc[MyData](
//	    func(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[MyData], []allocate.Item, float64, float64, error) {
//	        // 从数据库加载
//	        nodes := queryNodes(bizID, bizVersion)
//	        items := queryItems(bizID, bizVersion)
//	        rate, ratio := queryExtra(bizID, bizVersion)
//	        return nodes, items, rate, ratio, nil
//	    },
//	))
func WithDataLoader[T any](loader DataLoader[T]) Option[T] {
	return func(c *Calculator[T]) {
		c.dataLoader = loader
	}
}

// WithDataLoaderFunc 设置函数式数据加载器（简化版）
func WithDataLoaderFunc[T any](fn func(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[T], []Item, float64, float64, error)) Option[T] {
	return func(c *Calculator[T]) {
		c.dataLoader = DataLoaderFunc[T](fn)
	}
}

// WithItems 设置待分摊项列表（可选，覆盖 dataLoader 加载的 items）
func WithItems[T any](items []Item) Option[T] {
	return func(c *Calculator[T]) {
		c.items = items
	}
}

// WithExtra 设置加成比例和比例因子（可选，覆盖 dataLoader 加载的配置）
// rate: 加成比例（如 0.13 表示 13%）
// ratio: 加成比例因子（如税率），用于计算含比例总量
func WithExtra[T any](rate, ratio float64) Option[T] {
	return func(c *Calculator[T]) {
		c.extraRate = rate
		c.extraRatio = ratio
	}
}

// WithWeightFn 设置权重计算函数
// 叶子节点：返回 unitValue * quantity
// 中间节点：返回子节点权重和 * max(quantity, 1)
func WithWeightFn[T any](fn func(*tree.Node[T]) float64) Option[T] {
	return func(c *Calculator[T]) {
		c.weightFn = fn
	}
}

// WithRatioFn 设置比例因子获取函数
// 用于从节点数据中提取比例（如税率），计算净值
func WithRatioFn[T any](fn func(*tree.Node[T]) float64) Option[T] {
	return func(c *Calculator[T]) {
		c.ratioFn = fn
	}
}

// WithCache 设置缓存（可选，默认使用内存缓存）
func WithCache[T any](cache Cache) Option[T] {
	return func(c *Calculator[T]) {
		c.cache = cache
	}
}

// WithCacheTTL 设置缓存 TTL
func WithCacheTTL[T any](ttl time.Duration) Option[T] {
	return func(c *Calculator[T]) {
		c.cacheTTL = ttl
	}
}
