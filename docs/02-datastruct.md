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

## tree — 扁平列表 → 树（泛型，10 万节点 O(N)）

**类型**：`Node[T]`（泛型节点）、`SafeListStore[T]`、`Elem = Node[Empty]`（无扩展字段兼容别名）

**Node[T] 字段**：`ID / Pid / Name / Path / Ppath / Data T / Child`。业务扩展字段全部塞 `Data T`，类型安全零反射。`Path="/1/2/3/"`，`Ppath="/1/2/"`；根节点 `Ppath="/"`。

**无状态建树函数**：

| API | 复杂度 | 说明 |
|---|---|---|
| `BuildByPath[T](items) []*Node[T]` | O(N) | 按 Path/Ppath 建森林（数据行有 path 字段时首选） |
| `BuildByPid[T](items) []*Node[T]`  | O(N) | 按 ID/Pid 建森林 |
| `FindSubTree[T](roots, matchFn)` | O(N) | 找第一个匹配子树 |
| `Walk[T](roots, fn)` | O(N) | 前序遍历，fn 返 false 中止 |
| `Map[Src,T](src, convert)` | O(N) | model → Node[T] 一步映射 |

**SafeListStore[T]**（并发采集分页数据后统一建树）：`NewSafeListStore[T]() / Add / Len / GetAll / BuildTreeByPath / BuildTreeByPid / BuildTreeByCondition`。

```go
// 业务扩展字段
type BomPayload struct {
    Price      int64
    Supplier   string
    Level      int32
    HasChild   bool
}

// 一步 model → 节点
nodes := tree.Map(rows, func(r *model.TBomEdgeVersion) tree.Node[BomPayload] {
    return tree.Node[BomPayload]{
        ID: strconv.FormatInt(r.ID, 10), Pid: strconv.FormatInt(r.Pnid, 10),
        Name: r.Name, Path: r.Path, Ppath: r.Ppath,
        Data: BomPayload{Price: r.Price, Supplier: r.SupplierID, Level: r.Level},
    }
})
roots := tree.BuildByPath(nodes)

// 遍历读扩展字段
tree.Walk(roots, func(n *tree.Node[BomPayload]) bool {
    fmt.Println(n.Name, n.Data.Price)
    return true
})
```

无扩展字段直接用 `tree.Elem`（`= Node[struct{}]`）：

```go
roots := tree.BuildByPath([]tree.Elem{{ID:"1", Path:"/1/", Ppath:"/"}, ...})
```

**性能**：10 万节点建树 ≈ 50–70 ms/次（Data 字段大小相关）。

**迁移自** idx `internal/logic/bomscheme/bomSchemeNodeTreeLogic.go` 的 `BuildTreeFromEdges` pathMap 单次线性遍历思路，泛型化后适配任意业务节点。

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
