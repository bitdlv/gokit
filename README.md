# gokit

[![Go Reference](https://pkg.go.dev/badge/github.com/bitdlv/gokit.svg)](https://pkg.go.dev/github.com/bitdlv/gokit)
[![Go Report Card](https://goreportcard.com/badge/github.com/bitdlv/gokit)](https://goreportcard.com/report/github.com/bitdlv/gokit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.23-blue)](go.mod)

> **A comprehensive Go toolkit for building production-grade microservices.**
> 统一的 Go 公共库：覆盖认证、缓存、并发、数据访问、错误码、网关、文件处理、ID 生成等 42 个高质量子包。

**English** | [中文](#中文说明)

---

## ✨ Features

- **Modular design** — 42 focused packages, pick what you need
- **Production-ready** — battle-tested in real-world microservices
- **Zero lock-in** — minimal external deps, all features optional
- **Type-safe** — generics-based APIs, no `interface{}` abuse
- **Well-tested** — comprehensive test coverage in each package

### Highlights

| Category | Key Packages |
|----------|-------------|
| 🔐 **Auth & Security** | `jwt` (HS/RS/PS/ES/EdDSA), `middleware` (JWT auth middleware), `sign`, `cryptos`, `masking` |
| 💾 **Data Access** | `dbs` (GORM helpers), `cache` (Redis), `es` (Elasticsearch), `storage`, `idgen` (Snowflake/Segment) |
| 🚀 **Microservice** | `kit` (grpc-gateway helpers), `nacos`, `pb`, `httpx`, `ws`, `result` |
|| 📦 **Data Structures** | `tree` (generic tree builder), `chain`, `rank`, `paging` (concurrent pagination), `allocate` (tree-based cost allocation) |
| ⚡ **Concurrency** | `goroutinepool`, `scheduler`, `locker` (Redis lock) |
| 📝 **Utilities** | `excel` (read/write), `fileutils`, `time`, `random`, `conv`, `validator` |
| 🛠️ **Error Handling** | `errx` (error codes + i18n), `result` (unified response) |
| 📧 **Messaging** | `mail`, `msgnotice`, `pushgateway`, `senders` |

---

## 🚀 Quick Start

```bash
go get github.com/bitdlv/gokit
```

```go
import (
    "github.com/bitdlv/gokit/errx"
    "github.com/bitdlv/gokit/dbs"
    "github.com/bitdlv/gokit/cache"
    "github.com/bitdlv/gokit/jwt"
)
```

---

## 📚 Documentation

Detailed usage guides for each package:

| # | Category | Doc | Packages |
|---|----------|-----|----------|
| 1 | **Basics & Encoding** | [docs/01-basics.md](docs/01-basics.md) | conv, convert, cryptos, sign, random, masking, types, time |
|| 2 | **Data Structures** | [docs/02-datastruct.md](docs/02-datastruct.md) | chain, tree, rank, tryutils, **allocate** |
| 3 | **Concurrency** | [docs/03-concurrency.md](docs/03-concurrency.md) | goroutinepool, scheduler, locker |
| 4 | **Data Access** | [docs/04-data-access.md](docs/04-data-access.md) | dbs, cache, es, storage, idgen |
| 5 | **HTTP / RPC / Middleware** | [docs/05-http-rpc.md](docs/05-http-rpc.md) | httpx, ws, middleware, validator, result |
| 6 | **Error Codes** | [docs/06-errors.md](docs/06-errors.md) | errx, errx/legacy |
| 7 | **Microservice & Gateway** | [docs/07-microservice.md](docs/07-microservice.md) | kit, nacos, pb |
| 8 | **Messaging & Notification** | [docs/08-messaging.md](docs/08-messaging.md) | mail, msgnotice, pushgateway, senders |
| 9 | **Files & Documents** | [docs/09-files.md](docs/09-files.md) | excel, fileutils |
| 10 | **Runtime / Logging / System** | [docs/10-runtime.md](docs/10-runtime.md) | app, amslogx, instrumentation, ssh |
| 11 | **Helpers & Utilities** | [docs/11-helper.md](docs/11-helper.md) | helper (incl. clone, testconfig) |

---

## 🔍 Package Highlights

### `allocate` — Tree-based Cost Allocation with DataLoader
Generic tree structure value allocation with automatic data loading, version-based caching, and CRUD operations.

```go
import "github.com/bitdlv/gokit/allocate"

// 1. Implement DataLoader (auto-load by bizID + version)
loader := allocate.DataLoaderFunc[CostData](
    func(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[CostData], []allocate.Item, float64, float64, error) {
        return queryNodes(bizID, bizVersion), queryItems(bizID, bizVersion), 0.13, 0.13, nil
    },
)

// 2. Create Calculator (no manual nodes needed)
calc := allocate.NewCalculator(
    allocate.WithCacheKey[CostData]("project:123", 1),
    allocate.WithDataLoader[CostData](loader),
    allocate.WithWeightFn(func(n *tree.Node[CostData]) float64 {
        if len(n.Child) == 0 {
            return float64(n.Data.Price) * n.Data.Qty
        }
        return 0
    }),
)

// 3. Auto-load data and calculate
result, _ := calc.Calculate()

// 4. CRUD auto-sync (version bump + cache invalidation)
calc.AddItem(allocate.Item{ID: "tax", Name: "Tax", Value: 200})
result2, _ := calc.Calculate()  // Auto-load new version data
```

**Key Features:**
- **DataLoader auto-load**: No manual `nodes` preparation, auto-load by `bizID+bizVersion`
- **Version-based cache**: `allocate:{bizID}:v{bizVersion}`, zero CPU overhead
- **CRUD auto-sync**: Add/Remove/Update operations auto-increment version and invalidate cache
- **Multi-user share**: Same `bizID+version` hits same cache, version isolation ensures consistency
- **Largest remainder method**: Exact allocation without cumulative errors

### `allocate` — 树形数值分摊（DataLoader 自动加载）
基于泛型树结构的通用数值分摊，支持自动数据加载、版本缓存和 CRUD 操作。

```go
import "github.com/bitdlv/gokit/allocate"

// 1. 实现数据加载器（按 bizID+version 自动加载）
loader := allocate.DataLoaderFunc[CostData](
    func(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[CostData], []allocate.Item, float64, float64, error) {
        return queryNodes(bizID, bizVersion), queryItems(bizID, bizVersion), 0.13, 0.13, nil
    },
)

// 2. 创建计算器（无需手动传入节点数据）
calc := allocate.NewCalculator(
    allocate.WithCacheKey[CostData]("project:123", 1),
    allocate.WithDataLoader[CostData](loader),
    allocate.WithWeightFn(func(n *tree.Node[CostData]) float64 {
        if len(n.Child) == 0 {
            return float64(n.Data.Price) * n.Data.Qty
        }
        return 0
    }),
)

// 3. 自动加载数据并计算
result, _ := calc.Calculate()

// 4. CRUD 自动同步（版本递增 + 缓存失效）
calc.AddItem(allocate.Item{ID: "tax", Name: "税费", Value: 200})
result2, _ := calc.Calculate()  // 自动加载新版本数据
```

**核心特性：**
- **DataLoader 自动加载**：无需手动准备 `nodes`，按 `bizID+bizVersion` 自动加载
- **业务版本缓存**：`allocate:{bizID}:v{bizVersion}`，零 CPU 开销
- **CRUD 自动同步**：增删改操作自动递增版本号并失效缓存
- **多用户共享**：相同 `bizID+version` 命中同一缓存，版本隔离确保一致性
- **最大余数法**：精确分摊，无累加误差

### `jwt` — JWT 签名与验证
Full support for HS256/384/512, RS256/384/512, PS256/384/512, ES256/384/512, EdDSA.

```go
import "github.com/bitdlv/gokit/jwt"

// Sign (HS256)
token, _ := jwt.GenerateHS256("secret", time.Now().Unix(), 7200,
    jwt.KV("userId", "1001"),
)

// Parse
claims, _ := jwt.ValidateHS256(token, "secret")
fmt.Println(claims.UserID)
```

### `tree` — Generic Tree Builder
O(N) tree construction from flat slices, supports `path`-based and `pid`-based strategies.

```go
import "github.com/bitdlv/gokit/tree"

nodes := []Node{{ID: 1, Pid: 0}, {ID: 2, Pid: 1}, {ID: 3, Pid: 1}}
root := tree.BuildByPid(nodes, func(n Node) int64 { return n.ID }, ...)
```

### `idgen` — Distributed ID Generators
Snowflake / Segment / DateCode / HierCode / Serial — all goroutine-safe.

```go
import "github.com/bitdlv/gokit/idgen"

gen := idgen.NewSnowflake(1)
id := gen.Next() // int64
```

### `paging` — Concurrent Pagination Fetcher
Fetch all pages from a paginated API concurrently.

```go
import "github.com/bitdlv/gokit/paging"

all, _ := paging.FetchAll(ctx, 100, func(page, size int) ([]Item, int, error) {
    return fetchPageFromAPI(page, size)
})
```

### `errx` — Error Code Management
Unified error codes with i18n support.

```go
import "github.com/bitdlv/gokit/errx"

err := errx.New(errx.USER_NOT_FOUND)
return result.Error(err)
```

### `middleware` — HTTP Middlewares
JWT auth, header-to-context passthrough, response masking.

```go
import "github.com/bitdlv/gokit/middleware"

jwtAuth := middleware.NewJwtAuth(cfg, userLoader)
http.Handle("/api", jwtAuth.Handle(yourHandler))
```

### `excel` — Excel Read/Write
High-level API based on `excelize`.

```go
import "github.com/bitdlv/gokit/excel"

rows, _ := excel.ReadFile("data.xlsx", "Sheet1")
excel.WriteFile("out.xlsx", headers, rows)
```

### `helper/clone` — Deep Copy with Field Filtering
Struct cloning with `clone:"ignore"` tag support, gorm primaryKey detection, nested pointer handling.

```go
import "github.com/bitdlv/gokit/helper/clone"

dst := clone.Struct(src, clone.WithIgnoreFields("Password"))
```

---

## 📦 Installation

```bash
go get github.com/bitdlv/gokit@latest
```

---

## 🧪 Testing

```bash
go build ./...
go vet ./...
go test ./...
```

---

## 📄 License

[MIT](LICENSE)

---

## 🔗 Links

- **GoDoc**: https://pkg.go.dev/github.com/bitdlv/gokit
- **Issues**: https://github.com/bitdlv/gokit/issues
- **Changelog**: [CHANGELOG.md](CHANGELOG.md) (if exists)

---

## 中文说明

统一 Go 公共库：`github.com/bitdlv/gokit`

共 **43 个顶级包**，按用途划分为十一大类。所有包均通过 `go build ./...` / `go vet ./...` / `go mod tidy` 验证。

### 快速开始

```bash
go get github.com/bitdlv/gokit
```

```go
import (
    "github.com/bitdlv/gokit/errx"
    "github.com/bitdlv/gokit/dbs"
    "github.com/bitdlv/gokit/cache"
)
```

### 分类索引

详细使用说明见 [`docs/`](docs/README.md)。

| # | 分类 | 文档 | 包 |
|---|---|---|---|
| 一 | 基础数据 / 编码 | [docs/01-basics.md](docs/01-basics.md) | conv, convert, cryptos, sign, random, masking, types, time |
|| 二、数据结构/算法 | [02-datastruct.md](02-datastruct.md) | chain, tree, rank, tryutils, allocate |
| 三 | 并发 / 调度 | [docs/03-concurrency.md](docs/03-concurrency.md) | goroutinepool, scheduler, locker |
| 四 | 数据访问 | [docs/04-data-access.md](docs/04-data-access.md) | dbs, cache, es, storage, idgen |
| 五 | HTTP / RPC / 中间件 | [docs/05-http-rpc.md](docs/05-http-rpc.md) | httpx, ws, middleware, validator, result |
| 六 | 错误码 | [docs/06-errors.md](docs/06-errors.md) | errx, errx/legacy |
| 七 | 微服务 / 网关 | [docs/07-microservice.md](docs/07-microservice.md) | kit, nacos, pb |
| 八 | 消息 / 通知 | [docs/08-messaging.md](docs/08-messaging.md) | mail, msgnotice, pushgateway, senders |
| 九 | 文件 / 文档 | [docs/09-files.md](docs/09-files.md) | excel, fileutils |
| 十 | 运行时 / 日志 / 系统 | [docs/10-runtime.md](docs/10-runtime.md) | app, amslogx, instrumentation, ssh |
| 十一 | 杂项胶水 | [docs/11-helper.md](docs/11-helper.md) | helper（含 testconfig 子包） |

### 结构变更历史（v1 → v2）

| 变更 | 说明 |
|---|---|
| `xerr/` → `errx/legacy/` | 子包隔离，避免与主 `errx` 符号冲突 |
| `gochan/` → `goroutinepool/dispatcher/` | 协程池合并 |
| `taskpool/`、`utils/goroutinePool/` 删除 | 重复实现 |
| `storage/deivers/` → `storage/drivers/` | 拼写修正 |
| `utils/*.go` → `helper/*.go` | 8 个文件扁平合并 |
| `utils/testConfig/` → `helper/testconfig/` | 子包命名规范化 |
| 新增 `idgen/` | 通用 ID 生成器：Snowflake / Segment / DateCode / HierCode / Serial |
| 新增 `allocate/` | 通用树形数值分摊：DataLoader 自动加载 + 业务版本缓存 |

顶级包数量：44 → **43**

### 外部依赖 mock 建议

| 包 | mock 方案 |
|---|---|
| cache / locker / idgen | miniredis |
| dbs | sqlmock 或 docker mysql |
| es | httptest fake ES |
| httpx / ws | httptest.NewServer |
| mail / msgnotice / pushgateway | mock `http.RoundTripper` |

### 验证

```bash
go build ./...
go vet ./...
go mod tidy
```

### 文档

- 总索引：[docs/README.md](docs/README.md)
- 分类详解：[docs/01-basics.md](docs/01-basics.md) ~ [docs/11-helper.md](docs/11-helper.md)
