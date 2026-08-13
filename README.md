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
| 📦 **Data Structures** | `tree` (generic tree builder), `chain`, `rank`, `paging` (concurrent pagination) |
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
| 2 | **Data Structures** | [docs/02-datastruct.md](docs/02-datastruct.md) | chain, tree, rank, tryutils |
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

### `jwt` — JWT Signing & Verification
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

共 **42 个顶级包**，按用途划分为十一大类。所有包均通过 `go build ./...` / `go vet ./...` / `go mod tidy` 验证。

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
| 二 | 数据结构 / 算法 | [docs/02-datastruct.md](docs/02-datastruct.md) | chain, tree, rank, tryutils |
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

顶级包数量：44 → **42**

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
