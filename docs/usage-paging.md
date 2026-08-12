# gokit paging 使用说明

`github.com/bitdlv/gokit/paging` 提供分页数据的**并发全量抓取**和**页码归一化**工具，
迁移自 `idx/internal/utils/page.go`，通用化后剥离到公共库。

---

## API

### `FetchAll[T]` — 并发抓全量分页数据

```go
func FetchAll[T any](
    ctx context.Context,
    pageSize int,
    maxConcurrency int,
    fetchFn FetchFunc[T],
) ([]T, error)

type PageResult[T any] struct {
    Items []T
    Total int64
}
type FetchFunc[T any] func(ctx context.Context, page, pageSize int) (*PageResult[T], error)
```

行为：
1. 同步取 page=1，读出 `Total`。
2. `Total==0` 或第 1 页已含全部数据 → 直接返回。
3. `math.Ceil(Total/pageSize)` 计算总页数，剩余页用 `errgroup` 并发抓取。
4. `maxConcurrency > 0` 时 `g.SetLimit(maxConcurrency)` 限流；任一页 error 立刻取消其他在途请求并返回错误。
5. 按页码顺序（1..N）扁平化合并，**顺序稳定**。

### `Normalize` — 页码 / pageSize 归一化

```go
func Normalize(page, pageSize int, defaultPageSize ...int) (page, size, offset int)
```

- `page <= 0 → 1`
- `pageSize <= 0 → defaultPageSize`（可选，默认 10）
- 返回 `offset = (page-1) * pageSize`

---

## 使用示例

### 1. 全量抓取下游分页接口

```go
import "github.com/bitdlv/gokit/paging"

type User struct{ ID int64; Name string }

users, err := paging.FetchAll[User](ctx, 100, 8,
    func(ctx context.Context, page, size int) (*paging.PageResult[User], error) {
        resp, err := bpmClient.ListUsers(ctx, &bpmpb.ListReq{Page: int32(page), PageSize: int32(size)})
        if err != nil {
            return nil, err
        }
        items := make([]User, 0, len(resp.Items))
        for _, u := range resp.Items {
            items = append(items, User{ID: u.Id, Name: u.Name})
        }
        return &paging.PageResult[User]{Items: items, Total: resp.Total}, nil
    },
)
```

要点：
- `pageSize=100` 是每页大小；`maxConcurrency=8` 是并发上限，按下游 QPS 调。
- 闭包内**必须**用参数里的 `ctx`（`gCtx`），才能在其他页失败时被 errgroup 取消。
- 泛型 `T` 决定返回切片类型；直接返回业务模型或简单 DTO 均可。

### 2. 请求参数归一化

```go
func (l *ListLogic) List(ctx context.Context, in *pb.ListReq) (*pb.ListResp, error) {
    page, size, offset := paging.Normalize(int(in.Page), int(in.PageSize))
    var rows []*model.Foo
    err := l.svcCtx.DB.WithContext(ctx).
        Offset(offset).Limit(size).Find(&rows).Error
    _ = page
    // ...
}
```

---

## 迁移路径（idx / tpl → gokit）

原 `idx/internal/utils/page.go`：

| 原 API | 新 API |
| --- | --- |
| `utils.FetchAll[T]` | `paging.FetchAll[T]` |
| `utils.PageResult[T]` | `paging.PageResult[T]` |
| `utils.FetchFunc[T]` | `paging.FetchFunc[T]` |
| `utils.PageSize(p, s)` | `paging.Normalize(p, s)` |

替换步骤：

```bash
# 1. import 替换
gofmt -r 'utils.FetchAll -> paging.FetchAll' -w path/to/file.go
# 或直接 sed / IDE 重命名，然后：
goimports -w path/to/file.go   # 会自动补 github.com/bitdlv/gokit/paging
# 2. 删除 idx/internal/utils/page.go
```

注意：`PageSize` → `Normalize` 名字变了，`sed -i 's/utils\.PageSize\b/paging.Normalize/g'` 一次到位。

---

## 相关文件

- `gokit/paging/fetch.go` — `FetchAll` / `PageResult` / `FetchFunc`
- `gokit/paging/page.go` — `Normalize`
- `gokit/paging/fetch_test.go` — 单测（顺序合并 / 首页即全量 / 空 Total / 错误传播 / Normalize）
