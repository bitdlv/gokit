# 二、数据结构 / 算法

## chain — 责任链

```go
c := chain.New(handler1, handler2, handler3)
c.Exec(ctx)   // 顺序执行，任一 Abort 即中断
```

| API | 说明 |
|---|---|
| `New(handlers...)` | 构造链 |
| `Next(ctx)` | 触发下一个 handler |
| `Abort()` | 中断后续 |
| `Exec(ctx)` | 从头执行 |
| `AppendResult(v)` | 追加中间结果 |
| `Results() []any` | 获取所有结果 |

**测试**：用 mock handler 断言执行顺序 + Abort 行为。

## tree — 扁平列表 → 树

**类型**：`Elem`（元素）、`SafeListStore`（并发安全池）

| API | 说明 |
|---|---|
| `Add(elem)` | 添加节点 |
| `GetAll() []*Elem` | 全部节点 |
| `BuildTreeByCondition(cond)` | 按 parentId 建树 |
| `NewSafeListStore()` | 构造并发安全列表 |

## rank — 排序 + 过滤 + 分页

**类型**：`Ordered`、`Result`

```go
r := rank.NewRanker(data)
page := r.Sort(cmp).RankWhere(pred).RankAndPaginateResults(page, size)
```

| API | 说明 |
|---|---|
| `NewRanker(data)` | 构造排序器 |
| `Sort(cmp)` | 排序 |
| `RankWhere(pred)` | 过滤 |
| `Data()` | 获取当前切片 |
| `RankAndPaginateResults(page, size)` | 分页组装 |

## tryutils — 失败重试

**类型**：`Func`

```go
err := tryutils.Do(func() error { return callAPI() }, 3)
if tryutils.IsMaxRetries(err) { /* 已达上限 */ }
```

## 测试

`go test -v ./chain/... ./tree/... ./rank/... ./tryutils/...`
