package rank

import (
	"fmt"
	"testing"
)

type Item struct {
	N string
	V int
}

func Test_RankBy(t *testing.T) {
	items := []Item{
		{"A", 10},
		{"B", 30},
		{"C", 20},
		{"D", 30},
	}
	
	// 按 V 降序
	r := NewRanker(items, func(it Item) int { return it.V }, false)
	r.Sort()
	fmt.Println("排序后：", r.Data())
	
	// 查询：N = "A" 的排名
	posA := r.RankWhere(func(it Item) bool { return it.N == "A" })
	posD := r.RankWhere(func(it Item) bool { return it.N == "D" })
	fmt.Println("N=A 的排名：", posA) // 4
	fmt.Println("N=D 的排名：", posD) // 2
}
