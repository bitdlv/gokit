// Package allocate 提供通用的数值分摊算法。
//
// 典型应用: 将一笔总量按业务权重分摊到一组元素上，要求:
//  1. 各元素分得的量之和严格等于 total（不多不少）
//  2. 在元素数量与 total 允许的范围内，尽量让每个权重>0 的元素都能分到
//  3. 分摊比例尽量接近权重比例
//
// 使用主流的"最大余数法 (Largest Remainder Method / Hamilton method)"，
// 该算法在选举席位分配、金额分摊等场景广泛使用，且不会出现累加误差。
package allocate

import (
	"math"
	"sort"
)

// Allocate 根据权重将整数 total 分摊给 elems 中的每个元素，
// 通过 setFn 把分到的量回写到 A 的对应字段/键上，返回 elems 自身。
//
//   - total:    待分摊的总量（建议以最小单位整数表示），>=0
//   - elems:    参与分摊的元素列表
//   - weightFn: 返回元素的权重值，要求 >=0；权重<=0 视为不参与分摊（分得 0）
//   - setFn:    将分得的量写回元素 setFn(a, share)；
//               若需修改字段，A 应为指针类型 (如 []*Item) 或 map 等引用类型
//
// 返回 elems 本身，便于链式调用。除 total<=0 外，∑share == total。
//
// 算法说明:
//
//	设 W = sum(w_i)，理想份额 ideal_i = total * w_i / W
//	先对每个元素分配 floor(ideal_i)，剩余 R = total - sum(floor_i)
//	将余数 (ideal_i - floor_i) 较大的前 R 个元素各加 1
//	→ 累计误差为 0，且分配最接近理想值
//
// 边界场景:
//
//   - total <= 0 或所有权重为 0 → setFn 全部以 0 回写
//   - elems 为空 → 直接返回
//
// 若希望"权重>0 的元素至少分到 1"，请使用 AllocateAtLeastOne。
func Allocate[A any](total int64, elems []A, weightFn func(A) float64, setFn func(A, int64)) []A {
	allocateInternal(total, elems, weightFn, setFn, false)
	return elems
}

// AllocateAtLeastOne 与 Allocate 相同，但会保证每个权重 > 0 的元素至少分到 1。
//
// 适用场景：总量很小但参与方较多，希望避免出现"分到 0"的极端情况。
//
// 实现策略：
//  1. 先给每个权重>0 的元素预分配 1
//  2. 剩余 (total - 预分配数) 按权重用最大余数法分配
//  3. 若 total < 权重>0 的元素数 → 降级为普通最大余数法（物理上无法人人有份）
func AllocateAtLeastOne[A any](total int64, elems []A, weightFn func(A) float64, setFn func(A, int64)) []A {
	allocateInternal(total, elems, weightFn, setFn, true)
	return elems
}

func allocateInternal[A any](total int64, elems []A, weightFn func(A) float64, setFn func(A, int64), atLeastOne bool) {
	n := len(elems)
	if n == 0 {
		return
	}
	shares := make([]int64, n) // 默认全 0

	if total <= 0 {
		writeBack(elems, shares, setFn)
		return
	}

	// 收集权重；记录权重>0 的索引
	weights := make([]float64, n)
	var totalW float64
	positives := make([]int, 0, n)
	for i, e := range elems {
		w := weightFn(e)
		if w < 0 || math.IsNaN(w) || math.IsInf(w, 0) {
			w = 0
		}
		weights[i] = w
		if w > 0 {
			totalW += w
			positives = append(positives, i)
		}
	}
	if totalW == 0 || len(positives) == 0 {
		writeBack(elems, shares, setFn)
		return
	}

	remaining := total

	// 策略一：保证 >0 元素至少分到 1
	if atLeastOne && remaining >= int64(len(positives)) {
		for _, i := range positives {
			shares[i] = 1
		}
		remaining -= int64(len(positives))
		if remaining == 0 {
			writeBack(elems, shares, setFn)
			return
		}
	}

	// 最大余数法分配 remaining
	type frac struct {
		idx       int
		remainder float64
	}
	fracs := make([]frac, 0, len(positives))
	var distributed int64
	for _, i := range positives {
		ideal := float64(remaining) * weights[i] / totalW
		floor := int64(math.Floor(ideal))
		shares[i] += floor
		distributed += floor
		fracs = append(fracs, frac{idx: i, remainder: ideal - float64(floor)})
	}

	leftover := remaining - distributed
	if leftover > 0 {
		// 余数从大到小排序，相同余数时权重大的优先（稳定排序）
		sort.SliceStable(fracs, func(a, b int) bool {
			if fracs[a].remainder != fracs[b].remainder {
				return fracs[a].remainder > fracs[b].remainder
			}
			return weights[fracs[a].idx] > weights[fracs[b].idx]
		})
		if leftover > int64(len(fracs)) {
			leftover = int64(len(fracs))
		}
		for i := int64(0); i < leftover; i++ {
			shares[fracs[i].idx]++
		}
	}

	writeBack(elems, shares, setFn)
}

func writeBack[A any](elems []A, shares []int64, setFn func(A, int64)) {
	for i, e := range elems {
		setFn(e, shares[i])
	}
}
