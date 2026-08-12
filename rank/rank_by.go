package rank

import "sort"

// 定义一个 Ordered 约束，支持常见可比较类型
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

// Ranker 排序器
type Ranker[T any, K Ordered] struct {
	data    []T
	keyFunc func(T) K
	less    func(a, b K) bool
}

// 创建一个 Ranker，asc=true 表示升序，false 表示降序
func NewRanker[T any, K Ordered](data []T, keyFunc func(T) K, asc bool) *Ranker[T, K] {
	cp := make([]T, len(data))
	copy(cp, data)
	r := &Ranker[T, K]{data: cp, keyFunc: keyFunc}
	if asc {
		r.less = func(a, b K) bool { return a < b }
	} else {
		r.less = func(a, b K) bool { return a > b }
	}
	return r
}

// 排序
func (r *Ranker[T, K]) Sort() {
	sort.Slice(r.data, func(i, j int) bool {
		return r.less(r.keyFunc(r.data[i]), r.keyFunc(r.data[j]))
	})
}

// 获取排序结果
func (r *Ranker[T, K]) Data() []T {
	return r.data
}

// RankWhere 返回第一个满足条件的元素排名 (1-based)，未找到返回 -1
func (r *Ranker[T, K]) RankWhere(pred func(T) bool) int {
	for i, v := range r.data {
		if pred(v) {
			return i + 1
		}
	}
	return -1
}
