# gokit 功能清单

统一公共库 `github.com/bitdlv/gokit`，共 **42 个顶级包**，按用途分为十一类。

- 构建：`go build ./...` ✅
- vet：`go vet ./...`（仅 cache/senders 2 处历史 fmt 警告）
- mod tidy：无变更

## 分类索引

| 分类 | 文档 | 包 |
|---|---|---|
| 一、基础数据/编码 | [01-basics.md](01-basics.md) | conv, convert, cryptos, sign, random, masking, types, time |
|| 二、数据结构/算法 | [02-datastruct.md](02-datastruct.md) | chain, tree, rank, tryutils, **allocate** |
| 三、并发/调度 | [03-concurrency.md](03-concurrency.md) | goroutinepool, scheduler, locker, paging |
| 四、数据访问 | [04-data-access.md](04-data-access.md) | dbs, cache, es, storage, idgen |
| 五、HTTP/RPC/中间件 | [05-http-rpc.md](05-http-rpc.md) | httpx, ws, middleware, validator, result |
| 六、错误码 | [06-errors.md](06-errors.md) | errx, errx/legacy |
| 七、微服务/网关 | [07-microservice.md](07-microservice.md) | kit, nacos, pb |
| 八、消息/通知 | [08-messaging.md](08-messaging.md) | mail, msgnotice, pushgateway, senders |
| 九、文件/文档 | [09-files.md](09-files.md) | excel, fileutils |
| 十、运行时/日志/系统 | [10-runtime.md](10-runtime.md) | app, amslogx, instrumentation, environmentvariable, ssh |
| 十一、杂项胶水 | [11-helper.md](11-helper.md) | helper（含 testconfig 子包） |

## 结构变更历史（v1 → v2）

| 变更 | 说明 |
|---|---|
| xerr/ → errx/legacy/ | 子包隔离，避免与主 errx 符号冲突 |
| gochan/ → goroutinepool/dispatcher/ | 协程池合并 |
| taskpool/、utils/goroutinePool/ 删除 | 重复实现 |
| storage/deivers/ → storage/drivers/ | 拼写修正 |
| utils/*.go → helper/*.go | 8 文件扁平合并 |
| utils/testConfig/ → helper/testconfig/ | 子包命名规范化 |

| 顶级包数量：44 → **43**（新增 `idgen` 通用 ID 生成器、`paging` 分页组件与 `allocate` 树形分摊） |

- 详细用法见 [usage-paging.md](usage-paging.md)、[usage-header-jwtauth.md](usage-header-jwtauth.md)
- **亮点包**：[`allocate`](../allocate/README.md) — 树形数值分摊（DataLoader 自动加载 + 业务版本缓存）

## 外部依赖 mock 建议

| 包 | mock 方案 |
|---|---|
| cache / locker / idgen | miniredis |
| dbs | sqlmock 或 docker mysql |
| es | httptest fake ES |
| httpx / ws | httptest.NewServer |
| mail / msgnotice / pushgateway | mock http.RoundTripper |
