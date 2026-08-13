package helper

import (
	"sort"
	"testing"
)

func TestSliceOperatorReturnOrder(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{3, 4, 5}
	eq := func(x, y int) bool { return x == y }

	union, intersection, diffAB, diffBA := SliceOperator(a, b, eq)

	sort.Ints(union)
	sort.Ints(intersection)
	sort.Ints(diffAB)
	sort.Ints(diffBA)

	// union 必须包含两集合全部元素（历史上此处曾与 intersection 互换）
	if want := []int{1, 2, 3, 4, 5}; len(union) != len(want) {
		t.Fatalf("union = %v, want %v", union, want)
	}
	for i, v := range union {
		if v != []int{1, 2, 3, 4, 5}[i] {
			t.Fatalf("union = %v, want %v", union, []int{1, 2, 3, 4, 5})
		}
	}
	if len(intersection) != 1 || intersection[0] != 3 {
		t.Fatalf("intersection = %v, want [3]", intersection)
	}
	if len(diffAB) != 2 || diffAB[0] != 1 || diffAB[1] != 2 {
		t.Fatalf("differenceAB = %v, want [1 2]", diffAB)
	}
	if len(diffBA) != 2 || diffBA[0] != 4 || diffBA[1] != 5 {
		t.Fatalf("differenceBA = %v, want [4 5]", diffBA)
	}
}

func TestSliceOperatorEmpty(t *testing.T) {
	eq := func(x, y int) bool { return x == y }

	union, intersection, diffAB, diffBA := SliceOperator([]int{}, []int{}, eq)
	if union != nil || intersection != nil || diffAB != nil || diffBA != nil {
		t.Fatalf("empty inputs should return all nil, got %v %v %v %v",
			union, intersection, diffAB, diffBA)
	}
}
