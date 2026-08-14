# allocate - 通用树形数值分摊包

## 功能

- **数值分摊**：将任意数值（金额、工时、资源等）按权重比例分摊到树形结构的各节点
- **整树加成**：对所有节点按统一比例加成
- **基数计算**：支持自定义权重计算公式，默认叶子节点 `unitValue * quantity`，中间节点 `sum(children) * max(quantity, 1)`
- **DataLoader 自动加载**：通过 `bizID+bizVersion` 自动从数据存储加载节点数据，无需手动传入
- **缓存支持**：内存缓存（默认）+ Redis 缓存（可选），基于 `bizID+bizVersion` 共享缓存
- **CRUD 操作**：支持树节点、分摊项、加成比例的增删改，自动递增版本号并失效缓存

## 核心概念

| 概念 | 说明 |
|------|------|
| **Item** | 待分摊项（如费用、任务、资源），有总量 Value 和比例因子 Ratio |
| **Weight** | 节点权重，用于计算分摊比例 |
| **BaseValue** | 基础值，权重转换后的实际值 |
| **Ratio** | 比例因子，用于拆分净值（如含税/未税） |
| **Extra** | 加成，基于净值按比例计算 |
| **DataLoader** | 数据加载器接口，根据 bizID+version 自动加载数据 |
| **bizID+bizVersion** | 业务标识 + 版本号，用于缓存隔离和多用户同步 |

## 核心设计：DataLoader 自动加载

`Calculator` 不再手动传入 `nodes`，而是通过 `WithDataLoader` 设置数据加载器，`Calculate()` 时自动根据 `bizID+bizVersion` 加载：

```go
// 1. 实现数据加载器
loader := allocate.DataLoaderFunc[CostData](
    func(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[CostData], []allocate.Item, float64, float64, error) {
        // 从数据库/Redis/任意存储加载
        nodes := queryNodes(bizID, bizVersion)
        items := queryItems(bizID, bizVersion)
        rate, ratio := queryExtra(bizID, bizVersion)
        return nodes, items, rate, ratio, nil
    },
)

// 2. 创建 Calculator（无需传入 nodes）
calc := allocate.NewCalculator(
    allocate.WithCacheKey("project:123", 1),  // 业务ID + 版本号
    allocate.WithDataLoader(loader),           // 数据加载器
    allocate.WithWeightFn(weightFn),
)

// 3. Calculate() 自动加载数据并计算
result, _ := calc.Calculate()

// 4. 数据变更后，更新版本号，自动加载新数据
calc.AddItem(newItem)  // 自动递增 version，下次 Calculate 加载新数据
result2, _ := calc.Calculate()
```

## 使用示例

### 1. 金额分摊（费用 + 加成）

```go
package main

import (
    "context"
    "fmt"
    "github.com/bitdlv/gokit/allocate"
    "github.com/bitdlv/gokit/tree"
)

type CostData struct {
    Price   int64   // 单价（分）
    Qty     float64 // 数量
    TaxRate float64 // 税率
}

// 模拟数据库
type DB struct{}

func (db *DB) Load(ctx context.Context, bizID string, version int64) ([]tree.Node[CostData], []allocate.Item, float64, float64, error) {
    nodes := []tree.Node[CostData]{
        {ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: CostData{Qty: 1}},
        {ID: "2", Pid: "1", Path: "/1/2/", Ppath: "/1/", Data: CostData{Qty: 2}},
        {ID: "3", Pid: "1", Path: "/1/3/", Ppath: "/1/", Data: CostData{Qty: 1}},
        {ID: "4", Pid: "2", Path: "/1/2/4/", Ppath: "/1/2/", Data: CostData{Price: 100, Qty: 1, TaxRate: 0.13}},
        {ID: "5", Pid: "2", Path: "/1/2/5/", Ppath: "/1/2/", Data: CostData{Price: 200, Qty: 1, TaxRate: 0.13}},
        {ID: "6", Pid: "3", Path: "/1/3/6/", Ppath: "/1/3/", Data: CostData{Price: 300, Qty: 1, TaxRate: 0.13}},
    }
    items := []allocate.Item{
        {ID: "shipping", Name: "运输费", Value: 1000, Ratio: 0.13},
        {ID: "packing", Name: "包装费", Value: 500, Ratio: 0.13},
    }
    return nodes, items, 0.13, 0.13, nil
}

func main() {
    db := &DB{}

    calc := allocate.NewCalculator(
        allocate.WithCacheKey[CostData]("project:123", 1),
        allocate.WithDataLoader[CostData](db),
        allocate.WithWeightFn(func(n *tree.Node[CostData]) float64 {
            if len(n.Child) == 0 {
                return float64(n.Data.Price) * n.Data.Qty
            }
            return 0
        }),
        allocate.WithRatioFn(func(n *tree.Node[CostData]) float64 {
            return n.Data.TaxRate
        }),
    )

    result, err := calc.Calculate()
    if err != nil {
        panic(err)
    }

    fmt.Printf("总基础值: %d 分\n", result.TotalBase)
    fmt.Printf("总分摊额: %d 分\n", result.TotalAlloc)
    fmt.Printf("总加成: %d 分\n", result.TotalExtra)
    fmt.Printf("总报价: %d 分\n", result.GrandTotal)

    printTree(result.Roots, 0)
}

func printTree(nodes []*allocate.NodeDetail, depth int) {
    indent := ""
    for i := 0; i < depth; i++ {
        indent += "  "
    }
    for _, n := range nodes {
        fmt.Printf("%s节点%s: 基础值=%d, 分摊=%d, 加成=%d, 总额=%d\n",
            indent, n.NodeID, n.BaseValue, n.AllocTotal, n.Extra, n.Total)
        printTree(n.Children, depth+1)
    }
}
```

### 2. 工时分摊

```go
type TaskData struct {
    Hours float64
    Count float64
}

loader := allocate.DataLoaderFunc[TaskData](
    func(ctx context.Context, bizID string, version int64) ([]tree.Node[TaskData], []allocate.Item, float64, float64, error) {
        nodes := loadTasks(bizID, version)
        items := []allocate.Item{{ID: "mgmt", Name: "管理费", Value: 100, Ratio: 0.0}}
        return nodes, items, 0.10, 0.0, nil
    },
)

calc := allocate.NewCalculator(
    allocate.WithCacheKey[TaskData]("project:456", 1),
    allocate.WithDataLoader[TaskData](loader),
    allocate.WithWeightFn(func(n *tree.Node[TaskData]) float64 {
        if len(n.Child) == 0 {
            return n.Data.Hours * n.Data.Count
        }
        return 0
    }),
)
```

### 3. 使用 Redis 缓存

```go
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
cache := allocate.NewRedisCache(rdb, "allocate:")

calc := allocate.NewCalculator(
    allocate.WithCacheKey[CostData]("project:123", 1),
    allocate.WithDataLoader[CostData](loader),
    allocate.WithWeightFn(weightFn),
    allocate.WithCache[CostData](cache),
    allocate.WithCacheTTL[CostData](2*time.Hour),
)

result, _ := calc.Calculate()
```

### 4. 动态修改（CRUD 操作）

```go
calc := allocate.NewCalculator(
    allocate.WithCacheKey[CostData]("project:123", 1),
    allocate.WithDataLoader[CostData](loader),
    allocate.WithWeightFn(weightFn),
)

// 添加节点
calc.AddNode(tree.Node[CostData]{ID: "7", ...}, "2")

// 更新节点
calc.UpdateNode("4", func(n *tree.Node[CostData]) {
    n.Data.Price = 150
})

// 删除节点（及其子树）
calc.RemoveNode("5")

// 添加分摊项
calc.AddItem(allocate.Item{ID: "tax", Name: "税费", Value: 200})

// 更新分摊项
calc.UpdateItem("shipping", func(item *allocate.Item) {
    item.Value = 1200
})

// 删除分摊项
calc.RemoveItem("packing")

// 修改加成
calc.SetExtra(0.15, 0.15)

// 移除加成
calc.RemoveExtra()

// 强制重新计算（重新从 DataLoader 加载）
result, _ := calc.Recalculate()
```

## API 参考

### Calculator[T]

| 方法 | 说明 |
|------|------|
| `NewCalculator(opts...)` | 创建计算器（无需传入 nodes） |
| `Calculate() (*Result, error)` | 执行计算（自动加载 + 缓存） |
| `Recalculate() (*Result, error)` | 强制重新加载并计算 |
| `Invalidate() error` | 使缓存失效 |
| `Reload() error` | 强制重新加载数据 |

### 树节点操作

| 方法 | 说明 |
|------|------|
| `AddNode(node, parentID) error` | 添加节点（自动失效缓存 + 递增版本） |
| `AddNodes(nodes) error` | 批量添加节点 |
| `RemoveNode(nodeID) error` | 删除节点及其子树 |
| `UpdateNode(nodeID, updateFn) error` | 更新节点数据 |

### 分摊项操作

| 方法 | 说明 |
|------|------|
| `AddItem(item) error` | 添加分摊项（自动失效缓存 + 递增版本） |
| `AddItems(items) error` | 批量添加分摊项 |
| `RemoveItem(itemID) error` | 删除分摊项 |
| `UpdateItem(itemID, updateFn) error` | 更新分摊项 |

### 加成比例操作

| 方法 | 说明 |
|------|------|
| `SetExtra(rate, ratio float64) error` | 设置加成比例和比例因子 |
| `RemoveExtra() error` | 移除加成（设为0） |

### Option[T]

| 函数 | 说明 |
|------|------|
| `WithCacheKey[T](bizID, bizVersion)` | **必须**。设置业务缓存键 |
| `WithDataLoader[T](loader)` | **必须**。设置数据加载器 |
| `WithDataLoaderFunc[T](fn)` | 函数式数据加载器（简化版） |
| `WithItems[T](items []Item)` | 覆盖 dataLoader 加载的分摊项 |
| `WithExtra[T](rate, ratio float64)` | 覆盖 dataLoader 加载的加成配置 |
| `WithWeightFn[T](fn)` | 设置权重计算函数 |
| `WithRatioFn[T](fn)` | 设置比例因子函数 |
| `WithCache[T](cache Cache)` | 设置缓存（默认内存缓存） |
| `WithCacheTTL[T](ttl)` | 设置缓存 TTL |

### DataLoader[T]

```go
type DataLoader[T any] interface {
    Load(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[T], []Item, float64, float64, error)
}

// 函数式简化实现
type DataLoaderFunc[T any] func(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[T], []Item, float64, float64, error)
```

### Result

| 字段 | 说明 |
|------|------|
| `TotalBase` | 总基础值 |
| `TotalAlloc` | 总分摊额 |
| `TotalExtra` | 总加成 |
| `GrandTotal` | 总报价 |
| `GrandTotalNet` | 净值总报价 |
| `Roots` | 树形明细 |
| `NodeIndex` | 平铺索引（map[string]*NodeDetail） |

### NodeDetail

| 字段 | 说明 |
|------|------|
| `NodeID` | 节点ID |
| `BaseWeight` | 计算权重 |
| `BaseValue` | 基础值 |
| `BaseValueNet` | 净值基础值 |
| `AllocList` | 分摊明细列表 |
| `AllocTotal` | 分摊总额 |
| `SubTotal` | 小计 = BaseValue + AllocTotal |
| `SubTotalNet` | 净值小计 |
| `Extra` | 加成量 |
| `Total` | 总量 |
| `TotalNet` | 净值总量 |
| `Children` | 子节点明细 |

## 多用户共享场景

### 问题

Web 服务中，用户A 修改数据后，用户B 如何看到最新结果？

```
时间线：
t1: 用户A 创建 Calculator(bizID=123, v1), Calculate() → 缓存结果
t2: 用户B 创建 Calculator(bizID=123, v1), Calculate() → 命中缓存 ✓
t3: 用户A 修改数据（AddItem/SetExtra）→ 自动递增到 v2
t4: 用户B 使用 v2 创建 Calculator → 看到新数据 ✓
```

### 解决方案

```go
// 用户A：计算并缓存
calcA := allocate.NewCalculator(
    allocate.WithCacheKey("project:123", 1),
    allocate.WithDataLoader(loader),
    allocate.WithCache(cache),
)
resultA, _ := calcA.Calculate()

// 用户B：相同 bizID+version，命中缓存
calcB := allocate.NewCalculator(
    allocate.WithCacheKey("project:123", 1),
    allocate.WithDataLoader(loader),
    allocate.WithCache(cache),
)
resultB, _ := calcB.Calculate()  // 直接返回缓存

// 用户A 修改数据（自动递增版本号到2）
calcA.AddItem(newItem)

// 用户B 更新版本号，看到新数据
calcB2 := allocate.NewCalculator(
    allocate.WithCacheKey("project:123", 2),  // 新版本
    allocate.WithDataLoader(loader),
    allocate.WithCache(cache),
)
resultB2, _ := calcB2.Calculate()  // 重新计算
```

### 完整多用户示例

```go
// 模拟 Web 服务
type CalcService struct {
    cache allocate.Cache
    db    *DB
}

func (s *CalcService) GetCalculation(ctx context.Context, bizID string, version int64) (*allocate.Result, error) {
    calc := allocate.NewCalculator(
        allocate.WithCacheKey[CostData](bizID, version),
        allocate.WithDataLoader[CostData](s.db),
        allocate.WithWeightFn(weightFn),
        allocate.WithCache[CostData](s.cache),
    )
    return calc.Calculate()
}

func (s *CalcService) AddItem(ctx context.Context, bizID string, version int64, item allocate.Item) error {
    calc := allocate.NewCalculator(
        allocate.WithCacheKey[CostData](bizID, version),
        allocate.WithDataLoader[CostData](s.db),
        allocate.WithWeightFn(weightFn),
        allocate.WithCache[CostData](s.cache),
    )
    // 执行修改（自动失效缓存 + 递增版本号）
    return calc.AddItem(item)
}
```

## 缓存与幂等性

### 业务ID + 业务版本（推荐）

使用 `WithCacheKey(bizID, bizVersion)` 设置业务缓存键，避免自动哈希的 CPU 开销：

```go
// 用户A：计算并缓存
calcA := allocate.NewCalculator(
    allocate.WithCacheKey("project:123", 1),
    allocate.WithDataLoader(loader),
    allocate.WithCache(cache),
)
result1, _ := calcA.Calculate()

// 用户B：相同 bizID+version，命中缓存
calcB := allocate.NewCalculator(
    allocate.WithCacheKey("project:123", 1),
    allocate.WithDataLoader(loader),
    allocate.WithCache(cache),
)
result2, _ := calcB.Calculate()  // 直接返回缓存

// 数据变更后，调用方递增版本号
calcC := allocate.NewCalculator(
    allocate.WithCacheKey("project:123", 2),  // 新版本
    allocate.WithDataLoader(loader),
    allocate.WithCache(cache),
)
result3, _ := calcC.Calculate()  // 重新计算
```

### 缓存键格式

```
allocate:{bizID}:v{bizVersion}

示例:
  allocate:project:123:v1
  allocate:order:456:v3
  allocate:calc:789:v10
```

### 版本号管理

- **调用方负责维护 bizVersion**：数据变更时递增
- **CRUD 操作自动递增本地版本**：`AddNode/RemoveItem/SetExtra` 等操作后 `bizVersion++`
- **多实例同步**：各实例使用相同 `bizID+bizVersion` 命中同一缓存

### 注意事项

1. **Calculator 非并发安全**：不要并发调用同一个 Calculator 实例的方法
2. **版本号一致性**：多用户场景下，各实例应从数据库读取最新版本号
3. **必须设置 DataLoader**：`WithDataLoader` 未设置时，`Calculate()` 返回错误

## 算法说明

### 最大余数法 (Largest Remainder Method)

保证分摊总额严格等于原始总额，无累加误差：

1. 计算理想份额：`ideal_i = total * weight_i / totalWeight`
2. 先分配 `floor(ideal_i)`
3. 剩余 `R = total - sum(floor_i)`，按余数大小依次分配

### 树形分摊流程

```
Step 1: 从 DataLoader 加载数据（bizID + bizVersion）
Step 2: 构建树（tree.BuildByPath）
Step 3: 后序遍历计算权重和基础值
Step 4: 根级占比计算（按权重比例）
Step 5: 每个 Item 递归分摊（从根到叶）
Step 6: 汇总计算 + 加成
Step 7: 写入缓存
```

## 注意事项

1. **权重函数必须设置**：默认权重为 1，可能不符合业务需求
2. **数量为 0 按 1 处理**：中间节点的 quantity 为 0 时自动按 1 计算
3. **DataLoader 必须设置**：未设置时 `Calculate()` 返回错误
4. **金额精度**：内部使用 `int64`（最小单位，如分），避免浮点误差
5. **CRUD 自动递增版本**：数据变更后自动失效缓存并递增 `bizVersion`
