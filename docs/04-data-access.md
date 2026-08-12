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

## 测试

```bash
# dbs：docker mysql 或 sqlmock
go test -v ./dbs/...

# cache / locker：miniredis
go test -v ./cache/... ./locker/...

# es：httptest fake ES server
go test -v ./es/...

# storage
go test -v ./storage/...
```
