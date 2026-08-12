// Package paging 提供分页数据抓取与页码归一化的通用工具。
//
// 典型场景：调用一个只暴露"页码 + pageSize"的下游接口，需要一次性拿全量数据。
// FetchAll 会先同步取第 1 页拿到 Total，再并发抓取剩余页，按页码顺序合并返回。
//
// 迁移自 idx/internal/utils/page.go，从项目中通用化剥离。
package paging

import (
	"context"
	"math"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/sync/errgroup"
)

// PageResult 是下游分页接口返回的标准结构。
type PageResult[T any] struct {
	Items []T   // 当前页的数据列表
	Total int64 // 列表总数据量
}

// FetchFunc 是调用方需实现的分页拉取闭包。
//
//	ctx      用于超时 / 取消传播
//	page     当前请求页码（从 1 开始）
//	pageSize 每页请求数量
type FetchFunc[T any] func(ctx context.Context, page, pageSize int) (*PageResult[T], error)

// FetchAll 并发抓取所有分页数据并按页码顺序合并返回。
//
// 参数：
//
//	ctx            上下文；任一分页失败会通过 errgroup 取消其余在途请求
//	pageSize       每页拉取数据量
//	maxConcurrency 最大并发协程数（<=0 表示不限制，避免打挂下游请合理设置）
//	fetchFn        实际执行请求的闭包
//
// 语义：
//   - 先同步取第 1 页拿 Total；Total==0 或第 1 页已含全部数据则直接返回。
//   - 剩余页并发抓取，任一页 error 立刻取消全部并返回该错误。
//   - 结果按页码顺序（page 1..N）扁平化后返回，顺序稳定。
func FetchAll[T any](ctx context.Context, pageSize int, maxConcurrency int, fetchFn FetchFunc[T]) ([]T, error) {
	// 1. 同步拉取第一页，获取 Total
	firstPage, err := fetchFn(ctx, 1, pageSize)
	if err != nil {
		return nil, err
	}
	if firstPage == nil || firstPage.Total == 0 || int64(len(firstPage.Items)) >= firstPage.Total {
		if firstPage == nil {
			return nil, nil
		}
		return firstPage.Items, nil
	}

	// 2. 计算总页数
	totalPages := int(math.Ceil(float64(firstPage.Total) / float64(pageSize)))

	// 3. 预分配二维切片，按页码索引存储，确保并发抓取后顺序稳定
	results := make([][]T, totalPages)
	results[0] = firstPage.Items

	// 4. errgroup 控制并发和上下文取消
	g, gCtx := errgroup.WithContext(ctx)
	if maxConcurrency > 0 {
		g.SetLimit(maxConcurrency)
	}

	// 5. 并发抓取第 2 页到最后一页
	for p := 2; p <= totalPages; p++ {
		p := p // Go 1.22 之前必须捕获
		g.Go(func() error {
			res, err := fetchFn(gCtx, p, pageSize)
			if err != nil {
				logx.WithContext(gCtx).Errorf("paging.FetchAll: 获取第 %d 页失败: %v", p, err)
				return err
			}
			if res != nil {
				results[p-1] = res.Items
			}
			return nil
		})
	}

	// 6. 等待完成
	if err := g.Wait(); err != nil {
		logx.WithContext(ctx).Errorf("paging.FetchAll: 并发分页抓取失败: %v", err)
		return nil, err
	}

	// 7. 扁平化合并
	allItems := make([]T, 0, firstPage.Total)
	for _, batch := range results {
		allItems = append(allItems, batch...)
	}
	return allItems, nil
}
