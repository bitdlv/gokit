package tree_test

import (
	"strconv"
	"testing"

	"github.com/bitdlv/gokit/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// —— 业务侧自定义扩展数据 ——
type bomPayload struct {
	Price    int64
	Supplier string
	Level    int32
}

func TestBuildByPath_WithGenericData(t *testing.T) {
	items := []tree.Node[bomPayload]{
		{ID: "1", Name: "A", Path: "/1/", Ppath: "/", Data: bomPayload{Price: 100, Level: 1}},
		{ID: "2", Pid: "1", Name: "B", Path: "/1/2/", Ppath: "/1/", Data: bomPayload{Price: 200, Level: 2}},
		{ID: "3", Pid: "1", Name: "C", Path: "/1/3/", Ppath: "/1/", Data: bomPayload{Price: 300, Level: 2, Supplier: "S1"}},
		{ID: "4", Pid: "2", Name: "D", Path: "/1/2/4/", Ppath: "/1/2/", Data: bomPayload{Price: 400, Level: 3}},
	}
	roots := tree.BuildByPath(items)
	require.Len(t, roots, 1)
	a := roots[0]
	assert.Equal(t, int64(100), a.Data.Price)
	assert.Len(t, a.Child, 2)
	// 验证扩展字段随节点保留
	var c *tree.Node[bomPayload]
	for _, ch := range a.Child {
		if ch.ID == "3" {
			c = ch
		}
	}
	require.NotNil(t, c)
	assert.Equal(t, "S1", c.Data.Supplier)
}

func TestBuildByPid_Empty(t *testing.T) {
	// 零扩展字段：用 tree.Elem = Node[Empty]
	items := []tree.Elem{
		{ID: "1"},
		{ID: "2", Pid: "1"},
		{ID: "3", Pid: "2"},
	}
	roots := tree.BuildByPid(items)
	require.Len(t, roots, 1)
	assert.Equal(t, "1", roots[0].ID)
	require.Len(t, roots[0].Child, 1)
	require.Len(t, roots[0].Child[0].Child, 1)
}

func TestMap_ConvertAndBuild(t *testing.T) {
	type row struct {
		ID, Pid int64
		Name    string
		Path    string
		Ppath   string
		Price   int64
	}
	rows := []row{
		{ID: 1, Name: "A", Path: "/1/", Ppath: "/", Price: 10},
		{ID: 2, Pid: 1, Name: "B", Path: "/1/2/", Ppath: "/1/", Price: 20},
	}
	nodes := tree.Map(rows, func(r row) tree.Node[bomPayload] {
		return tree.Node[bomPayload]{
			ID: strconv.FormatInt(r.ID, 10), Pid: strconv.FormatInt(r.Pid, 10),
			Name: r.Name, Path: r.Path, Ppath: r.Ppath,
			Data: bomPayload{Price: r.Price},
		}
	})
	roots := tree.BuildByPath(nodes)
	require.Len(t, roots, 1)
	assert.Equal(t, int64(20), roots[0].Child[0].Data.Price)
}

func TestSafeListStore_Generic(t *testing.T) {
	s := tree.NewSafeListStore[bomPayload]()
	s.Add([]tree.Node[bomPayload]{
		{ID: "1", Data: bomPayload{Price: 1}},
		{ID: "2", Pid: "1", Data: bomPayload{Price: 2}},
		{ID: "3", Pid: "2", Data: bomPayload{Price: 3}},
	})
	assert.Equal(t, 3, s.Len())
	got := s.BuildTreeByCondition(func(n tree.Node[bomPayload]) bool { return n.ID == "2" })
	require.NotNil(t, got)
	assert.Equal(t, int64(2), got.Data.Price)
	assert.Len(t, got.Child, 1)
}

func TestWalk_EarlyStop(t *testing.T) {
	items := []tree.Elem{
		{ID: "1"}, {ID: "2", Pid: "1"}, {ID: "3", Pid: "1"}, {ID: "4", Pid: "2"},
	}
	roots := tree.BuildByPid(items)
	var visited []string
	tree.Walk(roots, func(n *tree.Elem) bool {
		visited = append(visited, n.ID)
		return n.ID != "2" // 命中 "2" 后中止
	})
	// 应该访问了 1, 2 就停（不进入 2 的子节点 4，也不进入 3）
	assert.Equal(t, []string{"1", "2"}, visited)
}

func TestToNodes_And_BuildTreeByPath(t *testing.T) {
	type row struct {
		ID, Pid    int64
		Name       string
		Path, Ppat string
		Price      int64
	}
	rows := []*row{
		{ID: 1, Name: "A", Path: "/1/", Ppat: "/", Price: 10},
		{ID: 2, Pid: 1, Name: "B", Path: "/1/2/", Ppat: "/1/", Price: 20},
		{ID: 3, Pid: 2, Name: "C", Path: "/1/2/3/", Ppat: "/1/2/", Price: 30},
	}
	keyFn := func(r *row) tree.Keys {
		return tree.Keys{
			ID: strconv.FormatInt(r.ID, 10), Pid: strconv.FormatInt(r.Pid, 10),
			Name: r.Name, Path: r.Path, Ppath: r.Ppat,
		}
	}
	// ToNodes → BuildByPath
	nodes := tree.ToNodes(rows, keyFn)
	require.Len(t, nodes, 3)
	assert.Equal(t, rows[1], nodes[1].Data) // Data 就是原始 *row
	roots := tree.BuildByPath(nodes)
	require.Len(t, roots, 1)
	assert.Equal(t, int64(10), roots[0].Data.Price)

	// 一步到位
	roots2 := tree.BuildTreeByPath(rows, keyFn)
	require.Len(t, roots2, 1)
	require.Len(t, roots2[0].Child, 1)
	assert.Equal(t, int64(30), roots2[0].Child[0].Child[0].Data.Price)
}

// 10 万节点建树基准
func BenchmarkToNodesParallel_100k(b *testing.B) {
	type row struct{ ID, Pid int64; Name, Path, Ppath string }
	src := make([]*row, 0, 100_000)
	for r := 0; r < 100; r++ {
		rid := "r" + strconv.Itoa(r)
		src = append(src, &row{Name: rid, Path: "/" + rid + "/", Ppath: "/"})
		for i := 0; i < 999; i++ {
			cid := rid + "_" + strconv.Itoa(i)
			src = append(src, &row{Name: cid, Path: "/" + rid + "/" + cid + "/", Ppath: "/" + rid + "/"})
		}
	}
	keyFn := func(r *row) tree.Keys {
		return tree.Keys{ID: r.Name, Name: r.Name, Path: r.Path, Ppath: r.Ppath}
	}
	b.Run("serial", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = tree.ToNodes(src, keyFn)
		}
	})
	b.Run("parallel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = tree.ToNodesParallel(src, keyFn, 0, 0)
		}
	})
}

func TestToNodesParallel_OrderPreserved(t *testing.T) {
	src := make([]int, 10_000)
	for i := range src {
		src[i] = i
	}
	keyFn := func(v int) tree.Keys { return tree.Keys{ID: strconv.Itoa(v)} }
	out := tree.ToNodesParallel(src, keyFn, 512, 4)
	require.Len(t, out, 10_000)
	for i, n := range out {
		assert.Equal(t, strconv.Itoa(i), n.ID)
		assert.Equal(t, i, n.Data)
	}
}

// 10 万节点建树基准（旧）
func BenchmarkBuildByPath_100k(b *testing.B) {
	items := make([]tree.Node[bomPayload], 0, 100_000)
	for r := 0; r < 100; r++ {
		rid := "r" + strconv.Itoa(r)
		items = append(items, tree.Node[bomPayload]{ID: rid, Path: "/" + rid + "/", Ppath: "/"})
		for i := 0; i < 999; i++ {
			cid := rid + "_" + strconv.Itoa(i)
			items = append(items, tree.Node[bomPayload]{
				ID: cid, Pid: rid,
				Path:  "/" + rid + "/" + cid + "/",
				Ppath: "/" + rid + "/",
				Data:  bomPayload{Price: int64(i)},
			})
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		roots := tree.BuildByPath(items)
		if len(roots) != 100 {
			b.Fatal("bad roots:", len(roots))
		}
	}
}
