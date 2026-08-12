# 四、数据访问

## dbs — MySQL / GORM 多实例 + 读写分离

**类型**：`DBConf`、`MysqlConf`、`DBConnConfig`、`DBPoolStats`、`DBInstance`、`GormInitConf`

```go
dbs.Init(dbs.DBConf{
    Default: mainCfg,
    Extras:  map[string]dbs.MysqlConf{"report": rptCfg},
})
db  := dbs.Default()      // *gorm.DB
db2 := dbs.Get("report")  // 命名实例
stats := dbs.GetPoolStats()
```

| API | 说明 |
|---|---|
| `Init(cfg)` | 初始化全部实例 |
| `InitDB(cfg)` | 初始化单个 |
| `Default()` | 默认实例 |
| `Get(name)` | 命名实例 |
| `GetPoolStats()` | 连接池监控 |
| `Validate(cfg)` | 校验配置 |

支持 `gorm.io/plugin/dbresolver` 主从。

## cache — Redis 缓存组件

**类型**：`Cache`、`CacheGenerator`、`Namespace`、`Page`、`Rank`、`BloomFilter`、`DistributedLock`

### 通用缓存

```go
c := cache.NewCache(rds, "biz:")
c.Set(ctx, "k", v, ttl)
c.Get(ctx, "k", &v)
c.Add(ctx, "k", v)
c.Exists(ctx, "k")
```

### 内存缓存
`NewMemoryCache(cache.WithMemoryExpire(d))`

### 防击穿

```go
g := cache.NewCacheGenerator(c)
val, err := g.GenerateWithLock(ctx, key, ttl, func() (any, error) {
    return loadFromDB()
})
```

### 分布式锁 / 布隆 / 排行

- `NewDistributedLock(rds, key)` · `AcquireLock` / `ReleaseLock` / `RenewLock`
- `NewBloomFilter(rds, key, size)`
- `NewRank(rds, key)` · `AddOrUpdateScore` · `GetRankPage` · `GetUserRank`

- `MustRdsClient(cfg)` — 便捷 Redis client。

## es — Elasticsearch v7 + OTel

```go
client := es.MustNewEs(cfg)   // 内含 OTel RoundTrip
```

| API | 说明 |
|---|---|
| `NewEs(cfg)` | 构造 |
| `MustNewEs(cfg)` | 构造失败 panic |
| `RoundTrip(req)` | http.RoundTripper 装 tracing |

## storage — 文件存储抽象

**类型**：`Storage`、`Driver`、`DriverCanServeFile`、`Local`；子包 `storage/drivers`

```go
s := storage.NewStorage(
    storage.NewLocal("/data"),
    storage.WithUrlPrefix("/files"),
)
s.Put("a.png", multipartFile)
s.PutBytes("b.txt", []byte("hi"))
data, _ := s.GetBytes("b.txt")
url := s.Url("a.png")
```

| API | 说明 |
|---|---|
| `NewStorage(driver, opts...)` | 构造 |
| `NewLocal(root)` | 本地驱动 |
| `Put(path, file)` / `PutBytes(path, data)` | 写入 |
| `GetBytes(path)` | 读取 |
| `Has(path)` | 存在检查 |
| `Url(path)` | 生成 URL |
| `ServeFile(w, r, path)` | HTTP 服务 |
| `WithUrlPrefix(p)` | URL 前缀 Option |

## idgen — ID / 编码生成器

**类型**：`Snowflake`、`SegmentGenerator` + `SegmentManager`、`DateCodeGenerator`、`HierCodeGenerator`、`SerialGenerator`

依赖 `redis+gorm`（Snowflake 除外，无依赖）。所有依赖 DB 查询的生成器均通过回调注入 `QueryMax*Func`，与业务 model 解耦。

### Snowflake — 进程本地雪花算法

```go
g, _ := idgen.NewSnowflake()          // workerID 从 POD_NAME/hostname 派生
                                      // 默认纪元 2024-01-01（DefaultSnowflakeEpochMs）
id := g.NextID()                      // int64，时钟回拨>5ms 返回 0

// 自定义纪元（同一业务的所有实例必须一致）
g2, _ := idgen.NewSnowflake(idgen.WithEpoch(1704067200000))

// 兼容旧数据：显式退回 Unix 纪元（ID 数值巨大，不推荐新业务使用）
g3, _ := idgen.NewSnowflake(idgen.WithEpoch(0))
```

⚠️ **MySQL 主键选型警示**

Snowflake 有两个天然特性对 MySQL 主键不友好：

1. **数值巨大**：Unix 纪元下 ID 起点 ~7.5×10^18，接近 int64 上限，用于 `INT` 列直接溢出；`BIGINT` 列虽可容纳但索引 key 长度顶格。
2. **跨 worker 交错**：高位是时间戳但多实例并发时按 workerID 分片，B+ 树插入呈"交错追加"，页分裂放大写放大。

**规则**：

| 场景 | 推荐 |
|---|---|
| MySQL 自增主键（BIGINT / INT） | ✅ **SegmentGenerator**（值贴近 MAX(id)+1，天然连续） |
| 日志 / 追踪 / 消息 ID / 无索引成本的分布式唯一 ID | ✅ Snowflake |
| 存量已用 Snowflake 做主键 | 至少切到 `WithEpoch(DefaultSnowflakeEpochMs)`，未来空间从 ~50 年 → ~69 年 |
| INT 类型主键 | ❌ 不能用 Snowflake；用 SegmentGenerator |


### SegmentGenerator — Redis 分段发号

按表名/命名空间隔离；首次使用时对齐到 `SELECT MAX(id) FROM <name>`；本地缓冲当前段 + 剩余<20% 时异步预取下一段。

```go
mgr := idgen.NewSegmentManager(rdb, db, 1000)  // step=1000
g   := mgr.Get("t_bom_calc_item")
id, _  := g.Next(ctx)
ids, _ := g.NextN(ctx, 100)
```

### DateCodeGenerator — `PREFIX-YYYYMMDD-NNNNNN`

日期粒度独立计数，Redis key TTL 到当天 23:59:59；冷启动通过 `QueryDateMaxSeqFunc` 从 DB 恢复。

```go
g := idgen.NewDateCodeGenerator(rdb, "BID", "bid:code:", 6,
    func(ctx context.Context, prefix, dateStr string) (int64, error) {
        // SELECT code FROM t_bid WHERE code LIKE 'BID-20260416-%' ORDER BY code DESC LIMIT 1
        return maxSeqFromCode, nil
    },
)
code, _ := g.NextCode(ctx)             // BID-20260416-000001
```

### HierCodeGenerator — 三级分层编码

分类 `A001` / SPU `A001-0001` / SKU `A001-0001-00001`。层级>26 自动进位 AA/AB/...。

```go
g := idgen.NewHierCodeGenerator(rdb,
    idgen.HierCodeConfig{},           // 零值即 3/4/5 位宽 + DefaultLevelPrefix
    queryCategoryMaxSeq, querySpuMaxSeq, querySkuMaxSeq,
)
cat, seq, _ := g.NextCategoryCode(ctx, 1)    // A001, 1
spu, _      := g.NextSpuCode(ctx, "A001")    // A001-0001
sku, _      := g.NextSkuCode(ctx, "A001-0001") // A001-0001-00001
```

### SerialGenerator — `{code}.{NNNNNN}` 流水号

支持按 code 前缀路由到不同 QueryMax 回调（例如模块编码 vs 普通物料编码）。

```go
g := idgen.NewSerialGenerator(rdb, "bom:serial:", 6, defaultQueryMax).
    RegisterPrefixQuery("M-", moduleQueryMax)
full, _ := g.NextCode(ctx, "A001-0001")   // A001-0001.000001
```

## 测试

```bash
# dbs：docker mysql 或 sqlmock
go test -v ./dbs/...

# cache / locker / idgen（Segment/DateCode/Hier/Serial）：miniredis
go test -v ./cache/... ./locker/... ./idgen/...

# es：httptest fake ES server
go test -v ./es/...

# storage
go test -v ./storage/...
```
