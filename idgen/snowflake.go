// Package idgen 提供多种 ID / 编码生成器：
//
//   - Snowflake       进程本地雪花算法（无依赖）
//   - Segment         Redis 分段发号（int64 主键，按表名/命名空间隔离，支持异步预取）
//   - DateCode        BID-20260416-000001 类日期序号编码
//   - HierCode        A001 / A001-0001 / A001-0001-00001 三级分层编码
//   - Serial          code.000001 通用流水号（业务侧注入 QueryMax 回调）
//
// 所有依赖 DB 查询的生成器均通过回调注入 QueryMax*Func，避免耦合具体 GORM model。
//
// ⚠️ MySQL 主键选型指引：
//
//	Snowflake 生成的 ID 数值达到 10^18 量级、高位含时间戳、跨 worker 交错插入
//	→ 不适合 MySQL 自增主键：INT 立即溢出；BIGINT 索引膨胀 + 页分裂放大写放大。
//	MySQL 主键请优先使用 SegmentGenerator（值贴近 MAX(id)+1，天然连续递增）。
//	Snowflake 适合：日志/追踪 ID、消息 ID、无索引成本的分布式唯一 ID。
package idgen

import (
	"hash/crc32"
	"os"
	"sync"
	"time"
)

const (
	snowflakeWorkerBits   = 10 // Pod hash (0~1023)
	snowflakeSequenceBits = 12 // 每毫秒最多 4096 个 ID
	snowflakeMaxWorkerID  = -1 ^ (-1 << snowflakeWorkerBits)
	snowflakeMaxSequence  = -1 ^ (-1 << snowflakeSequenceBits)
	snowflakeTimeShift    = snowflakeWorkerBits + snowflakeSequenceBits
	snowflakeWorkerShift  = snowflakeSequenceBits

	// DefaultSnowflakeEpochMs 默认自定义纪元：2024-01-01 UTC (ms)。
	// 相对 Unix 纪元偏移可让 ID 值从 0 起，压缩数值范围，
	// BIGINT 剩余空间从 ~50 年提升到 ~69 年。
	DefaultSnowflakeEpochMs int64 = 1704067200000
)

// Snowflake 兼容 Twitter Snowflake 的本地 ID 生成器。
// workerID 由 POD_NAME 或 hostname 的 CRC32 派生（0~1023）。
type Snowflake struct {
	mu        sync.Mutex
	lastStamp int64 // 相对 epoch 的毫秒
	sequence  int64
	workerID  int64
	epochMs   int64 // 自定义纪元起点（Unix 毫秒）；0 表示使用 Unix 纪元（兼容旧行为）
}

// SnowflakeOption 构造选项。
type SnowflakeOption func(*Snowflake)

// WithEpoch 指定自定义纪元起点（Unix 毫秒）。
// 传 0 保留 Unix 纪元（生成的 ID 数值巨大，仅为兼容旧数据）。
// 推荐使用 DefaultSnowflakeEpochMs（2024-01-01）。
//
// 注意：同一业务的 Snowflake 实例必须使用一致的 epoch，否则 ID 有序性和唯一性无保证。
func WithEpoch(epochMs int64) SnowflakeOption {
	return func(s *Snowflake) { s.epochMs = epochMs }
}

// NewSnowflake 创建一个 Snowflake 生成器，workerID 从环境派生。
// 默认使用 DefaultSnowflakeEpochMs 作为纪元。
func NewSnowflake(opts ...SnowflakeOption) (*Snowflake, error) {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		hn, _ := os.Hostname()
		podName = hn
	}
	hash := int64(crc32.ChecksumIEEE([]byte(podName))) % (snowflakeMaxWorkerID + 1)
	if hash < 0 {
		hash = -hash
	}
	s := &Snowflake{workerID: hash, epochMs: DefaultSnowflakeEpochMs}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// NewSnowflakeWithWorker 直接指定 workerID（用于测试或显式配置）。
func NewSnowflakeWithWorker(workerID int64, opts ...SnowflakeOption) *Snowflake {
	if workerID < 0 {
		workerID = -workerID
	}
	s := &Snowflake{
		workerID: workerID % (snowflakeMaxWorkerID + 1),
		epochMs:  DefaultSnowflakeEpochMs,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// timestamp 返回相对 epoch 的毫秒；epochMs==0 时等价于 Unix 毫秒。
func (g *Snowflake) timestamp() int64 {
	return time.Now().UnixNano()/1e6 - g.epochMs
}

func (g *Snowflake) waitNextMillis(last int64) int64 {
	ts := g.timestamp()
	for ts <= last {
		time.Sleep(time.Microsecond)
		ts = g.timestamp()
	}
	return ts
}

// NextID 生成一个全局唯一、有序的 ID；时钟回拨超过 5ms 返回 0。
func (g *Snowflake) NextID() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	ts := g.timestamp()
	if ts < g.lastStamp {
		offset := g.lastStamp - ts
		if offset < 5 {
			ts = g.waitNextMillis(g.lastStamp)
		} else {
			return 0
		}
	}
	if ts == g.lastStamp {
		g.sequence = (g.sequence + 1) & snowflakeMaxSequence
		if g.sequence == 0 {
			ts = g.waitNextMillis(g.lastStamp)
		}
	} else {
		g.sequence = 0
	}
	g.lastStamp = ts
	return (ts << snowflakeTimeShift) | (g.workerID << snowflakeWorkerShift) | g.sequence
}
