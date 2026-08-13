// Package tree 提供通用树形结构构建工具（泛型版）。
//
// 设计目标：
//   - 单树规模 ≤ 10 万节点，O(N) 单次线性遍历完成建树。
//   - 用泛型 Node[T] 支持任意业务扩展字段（Data T），零反射、类型安全。
//   - 同时支持两种建树策略：
//     1) BuildByPath：数据已带 path/ppath 冗余字段（如 BOM 树形结构）—— 最快，天然处理"同 cnid 挂多处"。
//     2) BuildByPid ：只有 id/pid 关系，无 path 字段。
//
// 通用 pathMap 单次线性遍历树构建算法，泛型化后适配任意业务节点。
package tree

import (
	"runtime"
	"sync"
)

// Node 通用泛型树节点。业务扩展字段放 Data 里。
//
// 约定：
//   - ID / Pid  ：主键 + 父主键（字符串以兼容 int64/string 混合场景，业务侧自行 strconv）
//   - Path      ：从根到当前节点的绝对路径，形如 "/1/2/3/"（含首尾斜杠）
//   - Ppath     ：父路径 "/1/2/"；根节点为 "/"
//   - Data      ：业务扩展数据（价格、供应商、hasChildren 标记、原始行等），完全由调用方决定
//   - Child     ：子节点列表（BuildBy* 会填充）
//
// 用法：
//
//	type BomPayload struct { Price int64; Supplier string; Level int32 }
//	items := []tree.Node[BomPayload]{{ID:"1", Path:"/1/", Ppath:"/", Data: BomPayload{...}}, ...}
//	roots := tree.BuildByPath(items)
type Node[T any] struct {
	ID    string     `json:"id"`
	Pid   string     `json:"pid"`
	Name  string     `json:"name"`
	Path  string     `json:"path"`
	Ppath string     `json:"ppath"`
	Data  T          `json:"data,omitempty"`
	Child []*Node[T] `json:"child"`
}

// Empty 无扩展字段时的占位类型，等价于 Node[Empty] = 只带 ID/Pid/Name/Path/Ppath 的裸节点。
type Empty = struct{}

// Elem 兼容别名：等价于 Node[Empty]，为不需要扩展字段的场景保留旧名。
// 新代码推荐直接用 Node[T]。
type Elem = Node[Empty]

// ─────────────────────────────────────────────────────────────
// 建树 —— 无状态泛型函数，O(N) 单次遍历
// ─────────────────────────────────────────────────────────────

// BuildByPath 按 Path/Ppath 建森林。O(N)。
//
// 语义：
//   - 每个节点通过其 Ppath 定位父（pathMap[node.Ppath]）挂到父的 Child。
//   - 父不在 items 中的节点视为森林根。
//   - 不修改入参：内部按值拷贝为 *Node[T]。
//
// 泛型 T 由调用方指定，Data 字段随节点一起传递。
func BuildByPath[T any](items []Node[T]) []*Node[T] {
	if len(items) == 0 {
		return nil
	}
	pathMap := make(map[string]*Node[T], len(items))
	nodes := make([]*Node[T], len(items))
	for i := range items {
		n := items[i]
		n.Child = nil
		nodes[i] = &n
		if n.Path != "" {
			pathMap[n.Path] = nodes[i]
		}
	}
	roots := make([]*Node[T], 0)
	for _, n := range nodes {
		if parent, ok := pathMap[n.Ppath]; ok && parent != n {
			parent.Child = append(parent.Child, n)
		} else {
			roots = append(roots, n)
		}
	}
	return roots
}

// BuildByPid 按 ID/Pid 建森林。O(N)。
//
// 相比 BuildByPath 慢一点点（Pid 查找而已，仍是 O(N)），
// 用于数据行没有 path 冗余字段的场景。
func BuildByPid[T any](items []Node[T]) []*Node[T] {
	if len(items) == 0 {
		return nil
	}
	idMap := make(map[string]*Node[T], len(items))
	nodes := make([]*Node[T], len(items))
	for i := range items {
		n := items[i]
		n.Child = nil
		nodes[i] = &n
		if n.ID != "" {
			idMap[n.ID] = nodes[i]
		}
	}
	roots := make([]*Node[T], 0)
	for _, n := range nodes {
		if parent, ok := idMap[n.Pid]; ok && parent != n {
			parent.Child = append(parent.Child, n)
		} else {
			roots = append(roots, n)
		}
	}
	return roots
}

// FindSubTree 在已建好的森林里找到第一个匹配的节点及其子树。O(N)。
func FindSubTree[T any](roots []*Node[T], matchFn func(*Node[T]) bool) *Node[T] {
	for _, r := range roots {
		if r == nil {
			continue
		}
		if matchFn(r) {
			return r
		}
		if got := FindSubTree(r.Child, matchFn); got != nil {
			return got
		}
	}
	return nil
}

// Walk 前序遍历森林。fn 返回 false 中止整个遍历。
func Walk[T any](roots []*Node[T], fn func(*Node[T]) bool) {
	for _, r := range roots {
		if !walk(r, fn) {
			return
		}
	}
}

func walk[T any](n *Node[T], fn func(*Node[T]) bool) bool {
	if n == nil {
		return true
	}
	if !fn(n) {
		return false
	}
	for _, c := range n.Child {
		if !walk(c, fn) {
			return false
		}
	}
	return true
}

// Map 把 []Src 映射为 []Node[T]，业务侧一次性完成 model → 树节点转换。
//
// convert 需返回节点核心字段（ID/Pid/Name/Path/Ppath）+ 业务扩展 Data。
//
// 示例：
//
//	nodes := tree.Map(rows, func(r *model.TBomEdgeVersion) tree.Node[BomPayload] {
//	    return tree.Node[BomPayload]{
//	        ID: strconv.FormatInt(r.ID, 10), Pid: strconv.FormatInt(r.Pnid, 10),
//	        Name: r.Name, Path: r.Path, Ppath: r.Ppath,
//	        Data: BomPayload{Price: r.Price, Supplier: r.SupplierID, Level: r.Level},
//	    }
//	})
//	roots := tree.BuildByPath(nodes)
func Map[Src any, T any](src []Src, convert func(Src) Node[T]) []Node[T] {
	if len(src) == 0 {
		return nil
	}
	out := make([]Node[T], 0, len(src))
	for _, s := range src {
		out = append(out, convert(s))
	}
	return out
}

// Keys 是 ToNodes 的字段提取结果：把业务模型 T 里散落的主键/父键/路径映射到 Node 的核心字段。
// 只需返回字符串（业务侧自行 strconv），Node.Data 会保留原始 T 值。
type Keys struct {
	ID, Pid, Name, Path, Ppath string
}

// ToNodes 把 []T 转换为 []Node[T]，Data 字段直接放原始 T 值。
//
// 相比 Map 的差异：
//   - Map    ：调用方完整构造 Node[T]（Data 类型可与 Src 不同，做投影/裁剪）。
//   - ToNodes：调用方只回答"这条数据的 ID/Pid/Name/Path/Ppath 是什么"，Data 就是原对象本身，
//     适合"业务模型直接当树节点扩展数据"的最常见场景。
//
// 示例：
//
//	rows, _ := query.TBomEdgeVersion.Find()   // []*model.TBomEdgeVersion
//	nodes := tree.ToNodes(rows, func(r *model.TBomEdgeVersion) tree.Keys {
//	    return tree.Keys{
//	        ID:    strconv.FormatInt(r.ID, 10),
//	        Pid:   strconv.FormatInt(r.Pnid, 10),
//	        Name:  r.Name,
//	        Path:  r.Path,
//	        Ppath: r.Ppath,
//	    }
//	})
//	roots := tree.BuildByPath(nodes)
//	// 遍历时 n.Data 就是 *model.TBomEdgeVersion，任意字段都能读
func ToNodes[T any](src []T, keyFn func(T) Keys) []Node[T] {
	if len(src) == 0 {
		return nil
	}
	out := make([]Node[T], 0, len(src))
	for _, s := range src {
		k := keyFn(s)
		out = append(out, Node[T]{
			ID: k.ID, Pid: k.Pid, Name: k.Name,
			Path: k.Path, Ppath: k.Ppath,
			Data: s,
		})
	}
	return out
}

// ToNodesParallel 是 ToNodes 的并行版本：把 src 按 batchSize 切片，用 workers 个 goroutine 并行执行 keyFn。
//
// 何时用：
//   - keyFn 里做重计算（大量 strconv、反射、字符串拼装、正则），单线程 CPU 打满。
//   - src 规模较大（经验值 ≥ 5 万条）。
//   - 小于 5 万或 keyFn 只是几个字段拷贝 → 直接用 ToNodes，串行更快（省掉调度开销）。
//
// 参数：
//   - batchSize   ≤ 0 时按 workers 均分（min 1024）
//   - workers     ≤ 0 时取 runtime.GOMAXPROCS(0)
//
// 顺序保证：输出的 []Node[T] 顺序与 src 一一对应（不打乱）。keyFn 必须是幂等纯函数（无共享状态副作用）。
func ToNodesParallel[T any](src []T, keyFn func(T) Keys, batchSize, workers int) []Node[T] {
	n := len(src)
	if n == 0 {
		return nil
	}
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if batchSize <= 0 {
		batchSize = (n + workers - 1) / workers
		if batchSize < 1024 {
			batchSize = 1024
		}
	}
	// 小规模直接串行，避免调度开销
	if n <= batchSize || workers == 1 {
		return ToNodes(src, keyFn)
	}

	out := make([]Node[T], n)
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for start := 0; start < n; start += batchSize {
		end := start + batchSize
		if end > n {
			end = n
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(lo, hi int) {
			defer wg.Done()
			defer func() { <-sem }()
			for i := lo; i < hi; i++ {
				s := src[i]
				k := keyFn(s)
				out[i] = Node[T]{
					ID: k.ID, Pid: k.Pid, Name: k.Name,
					Path: k.Path, Ppath: k.Ppath,
					Data: s,
				}
			}
		}(start, end)
	}
	wg.Wait()
	return out
}

// BuildTreeByPathParallel 并行版一步到位：ToNodesParallel + BuildByPath。
func BuildTreeByPathParallel[T any](src []T, keyFn func(T) Keys, batchSize, workers int) []*Node[T] {
	return BuildByPath(ToNodesParallel(src, keyFn, batchSize, workers))
}

// BuildTreeByPidParallel 并行版一步到位：ToNodesParallel + BuildByPid。
func BuildTreeByPidParallel[T any](src []T, keyFn func(T) Keys, batchSize, workers int) []*Node[T] {
	return BuildByPid(ToNodesParallel(src, keyFn, batchSize, workers))
}

// BuildTreeByPath 一步到位：[]T → Node[T] 森林（按 path）。O(N)。
func BuildTreeByPath[T any](src []T, keyFn func(T) Keys) []*Node[T] {
	return BuildByPath(ToNodes(src, keyFn))
}

// BuildTreeByPid 一步到位：[]T → Node[T] 森林（按 pid）。O(N)。
func BuildTreeByPid[T any](src []T, keyFn func(T) Keys) []*Node[T] {
	return BuildByPid(ToNodes(src, keyFn))
}

// ─────────────────────────────────────────────────────────────
// SafeListStore —— 并发安全的扁平列表容器 + 建树入口
// ─────────────────────────────────────────────────────────────

// SafeListStore 并发安全的 Node[T] 列表容器。
// 适合"多协程并发采集分页数据，最后统一建树"的场景。
type SafeListStore[T any] struct {
	mu   sync.RWMutex
	data []Node[T]
}

// NewSafeListStore 构造。T 由调用方显式指定：
//
//	store := tree.NewSafeListStore[BomPayload]()
func NewSafeListStore[T any]() *SafeListStore[T] {
	return &SafeListStore[T]{}
}

// Add 批量追加（并发安全）。
func (s *SafeListStore[T]) Add(items []Node[T]) {
	if len(items) == 0 {
		return
	}
	s.mu.Lock()
	s.data = append(s.data, items...)
	s.mu.Unlock()
}

// Len 当前数据量。
func (s *SafeListStore[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// GetAll 返回副本，避免外部修改污染内部数据。
func (s *SafeListStore[T]) GetAll() []Node[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Node[T](nil), s.data...)
}

// BuildTreeByPath 按 path/ppath 建森林。O(N)。
func (s *SafeListStore[T]) BuildTreeByPath() []*Node[T] {
	return BuildByPath(s.GetAll())
}

// BuildTreeByPid 按 id/pid 建森林。O(N)。
func (s *SafeListStore[T]) BuildTreeByPid() []*Node[T] {
	return BuildByPid(s.GetAll())
}

// BuildTreeByCondition 建整棵森林（按 pid）后返回第一个匹配子树。O(N)。
// 若需按 path 建再匹配，直接用 tree.FindSubTree(store.BuildTreeByPath(), matchFn)。
func (s *SafeListStore[T]) BuildTreeByCondition(matchFn func(Node[T]) bool) *Node[T] {
	roots := s.BuildTreeByPid()
	return FindSubTree(roots, func(n *Node[T]) bool { return matchFn(*n) })
}
