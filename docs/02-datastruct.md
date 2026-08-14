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

## allocate — 通用树形数值分摊（⭐ 亮点包，DataLoader 自动加载）

> **亮点特性**：业务版本缓存 + CRUD 自动同步 + 多用户共享，生产级树形分摊解决方案。

**核心设计**：通过 `WithCacheKey(bizID, bizVersion)` + `WithDataLoader` 自动加载数据，相同业务版本共享缓存。

**类型**：`Calculator[T]`、`Item`、`NodeDetail`、`Result`、`DataLoader[T]`

**计算流程**：
1. `Calculate()` 自动调用 `DataLoader.Load(bizID, bizVersion)` 加载数据
2. 构建树（`tree.BuildByPath`）
3. 后序遍历计算权重（叶子=`weightFn(node)`，中间=`sum(children)*max(qty,1)`）
4. 每个 Item 按权重递归分摊（最大余数法）
5. 汇总 + 加成（基于净值）

**CRUD 自动同步**：`AddNode/RemoveNode/UpdateNode/AddItem/RemoveItem/UpdateItem/SetExtra/RemoveExtra` 自动递增 `bizVersion` 并失效缓存。

```go
// 1. 实现数据加载器
loader := allocate.DataLoaderFunc[CostData](
    func(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[CostData], []allocate.Item, float64, float64, error) {
        return queryNodes(bizID, bizVersion), queryItems(bizID, bizVersion), 0.13, 0.13, nil
    },
)

// 2. 创建 Calculator（无需传入 nodes）
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

// 3. Calculate() 自动加载数据并计算
result, _ := calc.Calculate()

// 4. CRUD 操作（自动递增版本号）
calc.AddItem(allocate.Item{ID: "tax", Name: "税费", Value: 200})
result2, _ := calc.Calculate()  // 自动加载新版本数据
```

**缓存键格式**：`allocate:{bizID}:v{bizVersion}`

**多用户同步**：用户A 修改后版本号递增，用户B 使用新版本号创建 Calculator 即可看到新数据。

**测试**：14 个用例全部通过（基础分摊、CRUD、多用户、幂等性）

**详见**：[allocate/README.md](../allocate/README.md)

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
| `Map[Src,T](src, convert)` | O(N) | 完整构造 Node[T]（Data 类型可与 Src 不同，做投影/裁剪） |
| `ToNodes[T](src, keyFn) []Node[T]` | O(N) | 业务模型直接当 Data，只回答五个 Key 字段（最常用） |
| `ToNodesParallel[T](src, keyFn, batchSize, workers)` | O(N/P) | ToNodes 并行版；keyFn 有重计算且 N ≥ 5 万时才用 |
| `BuildTreeByPath[T](src, keyFn) []*Node[T]` | O(N) | 一步到位 = ToNodes + BuildByPath |
| `BuildTreeByPid[T](src, keyFn) []*Node[T]` | O(N) | 一步到位 = ToNodes + BuildByPid |

**Keys**：`struct { ID, Pid, Name, Path, Ppath string }`，`ToNodes` 系列的字段映射结果。

**Map vs ToNodes 怎么选**：
- `Map`：Src → Node[T]，两个类型不同（如 model.Row → 精简 Payload），做类型投影/裁剪。
- `ToNodes`：Src 直接放进 `Data T`，业务模型就是节点扩展数据。最常见场景，写起来最省。

**SafeListStore[T]**（并发采集分页数据后统一建树）：`NewSafeListStore[T]() / Add / Len / GetAll / BuildTreeByPath / BuildTreeByPid / BuildTreeByCondition`。

```go
// 业务扩展字段
type BomPayload struct {
    Price    int64
    Supplier string
    Level    int32
    HasChild bool
}

// —— 方式 A：ToNodes（业务模型直接当 Data，最常用）
rows, _ := query.TBomEdgeVersion.Find() // []*model.TBomEdgeVersion
roots := tree.BuildTreeByPath(rows, func(r *model.TBomEdgeVersion) tree.Keys {
    return tree.Keys{
        ID:    strconv.FormatInt(r.ID, 10),
        Pid:   strconv.FormatInt(r.Pnid, 10),
        Name:  r.Name,
        Path:  r.Path,
        Ppath: r.Ppath,
    }
})
tree.Walk(roots, func(n *tree.Node[*model.TBomEdgeVersion]) bool {
    fmt.Println(n.Name, n.Data.Price) // 任意 model 字段直读
    return true
})

// —— 方式 B：Map（Src → Payload 类型投影）
nodes := tree.Map(rows, func(r *model.TBomEdgeVersion) tree.Node[BomPayload] {
    return tree.Node[BomPayload]{
        ID: strconv.FormatInt(r.ID, 10), Pid: strconv.FormatInt(r.Pnid, 10),
        Name: r.Name, Path: r.Path, Ppath: r.Ppath,
        Data: BomPayload{Price: r.Price, Supplier: r.SupplierID, Level: r.Level},
    }
})
roots2 := tree.BuildByPath(nodes)

// —— 方式 C：keyFn 内含重计算 + 大数据量 → ToNodesParallel
// workers=0 → GOMAXPROCS, batchSize=0 → 自动均分（min 1024）
nodes = tree.ToNodesParallel(rows, keyFn, 0, 0)
roots3 := tree.BuildByPath(nodes)
```

无扩展字段直接用 `tree.Elem`（`= Node[struct{}]`）：

```go
roots := tree.BuildByPath([]tree.Elem{{ID:"1", Path:"/1/", Ppath:"/"}, ...})
```

**性能基线（macOS i7-8850H，10 万节点）**：
- `BuildByPath` ≈ 21 ms/op
- `ToNodes` ≈ 7.4 ms/op；`ToNodesParallel` ≈ 4.4 ms/op（1.67×，keyFn 只做字段拷贝的最劣情况；keyFn 有 strconv/正则等重计算时加速比更高）
- **注意**：`BuildByPath` 本身没提供并行版——分桶+拷贝的开销比 O(N) map 查找还大，并行反而更慢。加速请集中在 `ToNodesParallel`（keyFn 层）。

基于 pathMap 单次线性遍历的通用树构建算法，泛型化后适配任意业务节点。

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
