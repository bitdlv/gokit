# gokit

统一 Go 公共库：`github.com/bitdlv/gokit`

共 **42 个顶级包**，按用途划分为十一大类。所有包均通过 `go build ./...` / `go vet ./...`（历史 fmt 警告 2 处）/ `go mod tidy` 验证。

## 快速开始

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

## 分类索引

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
| 十 | 运行时 / 日志 / 系统 | [docs/10-runtime.md](docs/10-runtime.md) | app, amslogx, instrumentation, environmentvariable, ssh |
| 十一 | 杂项胶水 | [docs/11-helper.md](docs/11-helper.md) | helper（含 testconfig 子包） |

## 结构变更历史（v1 → v2）

| 变更 | 说明 |
|---|---|
| `xerr/` → `errx/legacy/` | 子包隔离，避免与主 `errx` 符号冲突 |
| `gochan/` → `goroutinepool/dispatcher/` | 协程池合并 |
| `taskpool/`、`utils/goroutinePool/` 删除 | 重复实现 |
| `storage/deivers/` → `storage/drivers/` | 拼写修正 |
| `utils/*.go` → `helper/*.go` | 8 个文件扁平合并 |
| `utils/testConfig/` → `helper/testconfig/` | 子包命名规范化 |
| 新增 `idgen/` | 从 idx 项目 `internal/svc/*_gen.go` 抽象合并：Snowflake / Segment / DateCode / HierCode / Serial |

顶级包数量：44 → **42**

## 外部依赖 mock 建议

| 包 | mock 方案 |
|---|---|
| cache / locker / idgen | miniredis |
| dbs | sqlmock 或 docker mysql |
| es | httptest fake ES |
| httpx / ws | httptest.NewServer |
| mail / msgnotice / pushgateway | mock `http.RoundTripper` |

## 验证

```bash
go build ./...
go vet ./...
go mod tidy
```

## 文档

- 总索引：[docs/README.md](docs/README.md)
- 分类详解：[docs/01-basics.md](docs/01-basics.md) ~ [docs/11-helper.md](docs/11-helper.md)
