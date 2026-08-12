package tree

import (
	"fmt"
	"github.com/samber/lo"
	"strings"
	"testing"
)

func Test_Tree(t *testing.T) {
	tree := NewSafeListStore()
	tree.Add([]Elem{
		{ID: "1", Name: "园区", Pid: "0"},
		{ID: "1-2", Name: "楼宇", Pid: "1"},
		{ID: "1-2-3", Name: "楼层", Pid: "1-2"},
		{ID: "1-2-3-4", Name: "房间1", Pid: "1-2-3"},
		{ID: "1-2-3-5", Name: "房间2", Pid: "1-2-3"},

		{ID: "1-2-3-4-6", Name: "Node6", Pid: "1-2-3-4"},
		{ID: "1-2-3-4-7", Name: "Node7", Pid: "1-2-3-4"},
		{ID: "1-2-3-4-8", Name: "Node8", Pid: "1-2-3-4"},
		{ID: "1-2-3-4-9", Name: "Node9", Pid: "1-2-3-4"},
	})
	tt := tree.BuildTreeByCondition(func(i Elem) bool { return true })
	fmt.Println(tt)
}

func Test_Lo(t *testing.T) {
	type Item1 struct {
		Name string
		Id   int
	}

	type Item2 struct {
		Uid int32
		Age int
	}
	arr1 := []Item1{
		{Name: "zhang", Id: 1},
		{Name: "zhang", Id: 2},
		{Name: "wang", Id: 3},
	}
	aMap := lo.UniqBy(arr1, func(item Item1) string {
		return item.Name
	})
	str := " abc-def-gh-ijk-lmg"
	a := strings.Split(str, "-")
	res := strings.Join(a[:len(a)-1], "-")
	fmt.Println(res, aMap)
	// map[string][int]{ "apple":1, "banana":2 }
}
