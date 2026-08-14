package allocate_test

import (
	"context"
	"testing"

	"github.com/bitdlv/gokit/allocate"
	"github.com/bitdlv/gokit/tree"
)

// TestData 测试数据
type TestData struct {
	UnitValue int64
	Quantity  float64
	Ratio     float64
}

func (d TestData) GetQuantity() float64 { return d.Quantity }

// ─────────────────────────────────────────────
// 模拟数据存储
// ─────────────────────────────────────────────

// mockDataStore 模拟数据存储（如数据库）
type mockDataStore struct {
	nodes      []tree.Node[TestData]
	items      []allocate.Item
	extraRate  float64
	extraRatio float64
}

func (s *mockDataStore) Load(ctx context.Context, bizID string, bizVersion int64) ([]tree.Node[TestData], []allocate.Item, float64, float64, error) {
	return s.nodes, s.items, s.extraRate, s.extraRatio, nil
}

// ─────────────────────────────────────────────
// 基础算法测试
// ─────────────────────────────────────────────

func TestAllocate_Basic(t *testing.T) {
	type Slot struct {
		id    string
		share int64
	}
	slots := []*Slot{{id: "a"}, {id: "b"}, {id: "c"}}

	allocate.Allocate(100, slots, func(s *Slot) float64 {
		switch s.id {
		case "a":
			return 1
		case "b":
			return 2
		case "c":
			return 3
		}
		return 0
	}, func(s *Slot, v int64) { s.share = v })

	total := int64(0)
	for _, s := range slots {
		total += s.share
		t.Logf("%s: %d", s.id, s.share)
	}
	if total != 100 {
		t.Errorf("total = %d, want 100", total)
	}
}

func TestAllocate_AtLeastOne(t *testing.T) {
	type Slot struct {
		id    string
		share int64
	}
	slots := []*Slot{{id: "a"}, {id: "b"}, {id: "c"}}

	allocate.AllocateAtLeastOne(2, slots, func(s *Slot) float64 {
		return 1
	}, func(s *Slot, v int64) { s.share = v })

	total := int64(0)
	for _, s := range slots {
		total += s.share
		t.Logf("%s: %d", s.id, s.share)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
}

// ─────────────────────────────────────────────
// Calculator 测试
// ─────────────────────────────────────────────

func TestCalculator_BasicTree(t *testing.T) {
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 0, Quantity: 1}},
			{ID: "2", Pid: "1", Path: "/1/2/", Ppath: "/1/", Data: TestData{UnitValue: 0, Quantity: 2}},
			{ID: "3", Pid: "1", Path: "/1/3/", Ppath: "/1/", Data: TestData{UnitValue: 0, Quantity: 1}},
			{ID: "4", Pid: "2", Path: "/1/2/4/", Ppath: "/1/2/", Data: TestData{UnitValue: 100, Quantity: 1}},
			{ID: "5", Pid: "2", Path: "/1/2/5/", Ppath: "/1/2/", Data: TestData{UnitValue: 200, Quantity: 1}},
			{ID: "6", Pid: "3", Path: "/1/3/6/", Ppath: "/1/3/", Data: TestData{UnitValue: 300, Quantity: 1}},
		},
		items: []allocate.Item{
			{ID: "fee1", Name: "费用1", Value: 1000, Ratio: 0.13},
		},
		extraRate:  0.13,
		extraRatio: 0.13,
	}

	calc := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:tree", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			if len(n.Child) == 0 {
				return float64(n.Data.UnitValue) * n.Data.Quantity
			}
			return 0
		}),
		allocate.WithRatioFn(func(n *tree.Node[TestData]) float64 {
			return n.Data.Ratio
		}),
	)

	result, err := calc.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	t.Logf("TotalBase: %d", result.TotalBase)
	t.Logf("TotalAlloc: %d", result.TotalAlloc)
	t.Logf("TotalExtra: %d", result.TotalExtra)
	t.Logf("GrandTotal: %d", result.GrandTotal)

	if len(result.Roots) != 1 {
		t.Errorf("len(Roots) = %d, want 1", len(result.Roots))
	}
}

func TestCalculator_MultipleItems(t *testing.T) {
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 0, Quantity: 1}},
			{ID: "2", Pid: "1", Path: "/1/2/", Ppath: "/1/", Data: TestData{UnitValue: 100, Quantity: 2}},
			{ID: "3", Pid: "1", Path: "/1/3/", Ppath: "/1/", Data: TestData{UnitValue: 200, Quantity: 1}},
		},
		items: []allocate.Item{
			{ID: "shipping", Name: "运输费", Value: 1000, Ratio: 0.13},
			{ID: "packing", Name: "包装费", Value: 500, Ratio: 0.13},
			{ID: "tax", Name: "税费", Value: 300, Ratio: 0.0},
		},
		extraRate:  0.10,
		extraRatio: 0.0,
	}

	calc := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:multi", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			if len(n.Child) == 0 {
				return float64(n.Data.UnitValue) * n.Data.Quantity
			}
			return 0
		}),
	)

	result, err := calc.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	if result.TotalAlloc != 1800 {
		t.Errorf("TotalAlloc = %d, want 1800", result.TotalAlloc)
	}
}

func TestCalculator_Cache(t *testing.T) {
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 100, Quantity: 1}},
		},
	}

	cache := allocate.NewMemoryCache()

	// 第一次计算
	calc1 := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:cache", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)
	result1, err := calc1.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	// 第二次计算（相同 bizID+version，命中缓存）
	calc2 := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:cache", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)
	result2, err := calc2.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	if result1.GrandTotal != result2.GrandTotal {
		t.Errorf("cached result different: %d vs %d", result1.GrandTotal, result2.GrandTotal)
	}

	// 版本号变更，重新计算
	calc3 := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:cache", 2),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)
	result3, err := calc3.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	if result1.GrandTotal != result3.GrandTotal {
		t.Errorf("recalculated result different: %d vs %d", result1.GrandTotal, result3.GrandTotal)
	}
}

func TestCalculator_EdgeCases(t *testing.T) {
	t.Run("EmptyNodes", func(t *testing.T) {
		store := &mockDataStore{nodes: []tree.Node[TestData]{}}
		calc := allocate.NewCalculator(
			allocate.WithCacheKey[TestData]("test:empty", 1),
			allocate.WithDataLoader[TestData](store),
		)
		_, err := calc.Calculate()
		if err == nil {
			t.Error("expected error for empty nodes")
		}
	})

	t.Run("ZeroWeight", func(t *testing.T) {
		store := &mockDataStore{
			nodes: []tree.Node[TestData]{
				{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 0, Quantity: 0}},
			},
			items: []allocate.Item{{ID: "fee", Name: "费用", Value: 1000}},
		}
		calc := allocate.NewCalculator(
			allocate.WithCacheKey[TestData]("test:zero", 1),
			allocate.WithDataLoader[TestData](store),
			allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 { return 0 }),
		)
		result, err := calc.Calculate()
		if err != nil {
			t.Fatalf("Calculate() error = %v", err)
		}
		if result.TotalAlloc != 0 {
			t.Errorf("TotalAlloc = %d, want 0", result.TotalAlloc)
		}
	})

	t.Run("ZeroQuantity", func(t *testing.T) {
		store := &mockDataStore{
			nodes: []tree.Node[TestData]{
				{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 0, Quantity: 0}},
				{ID: "2", Pid: "1", Path: "/1/2/", Ppath: "/1/", Data: TestData{UnitValue: 100, Quantity: 1}},
			},
		}
		calc := allocate.NewCalculator(
			allocate.WithCacheKey[TestData]("test:zeroqty", 1),
			allocate.WithDataLoader[TestData](store),
			allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
				if len(n.Child) == 0 {
					return float64(n.Data.UnitValue) * n.Data.Quantity
				}
				return 0
			}),
		)
		result, err := calc.Calculate()
		if err != nil {
			t.Fatalf("Calculate() error = %v", err)
		}
		if result.TotalBase != 100 {
			t.Errorf("TotalBase = %d, want 100", result.TotalBase)
		}
	})
}

func TestCalculator_RatioSplit(t *testing.T) {
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 100, Quantity: 1, Ratio: 0.13}},
		},
		items: []allocate.Item{
			{ID: "fee", Name: "费用", Value: 113, Ratio: 0.13},
		},
	}

	calc := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:ratio", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithRatioFn(func(n *tree.Node[TestData]) float64 {
			return n.Data.Ratio
		}),
	)

	result, err := calc.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	root := result.Roots[0]
	t.Logf("BaseValue=%d, BaseValueNet=%d, AllocTotal=%d, SubTotal=%d, SubTotalNet=%d",
		root.BaseValue, root.BaseValueNet, root.AllocTotal, root.SubTotal, root.SubTotalNet)

	if root.BaseValue != 100 {
		t.Errorf("BaseValue = %d, want 100", root.BaseValue)
	}
	if root.AllocTotal != 113 {
		t.Errorf("AllocTotal = %d, want 113", root.AllocTotal)
	}
}

// ─────────────────────────────────────────────
// CRUD 操作测试
// ─────────────────────────────────────────────

func TestCalculator_NodeCRUD(t *testing.T) {
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 100, Quantity: 1}},
		},
	}

	calc := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:crud", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
	)

	// 第一次计算
	result1, err := calc.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	if result1.TotalBase != 100 {
		t.Errorf("TotalBase = %d, want 100", result1.TotalBase)
	}

	// 添加节点
	newNode := tree.Node[TestData]{
		ID: "2", Pid: "1", Path: "/1/2/", Ppath: "/1/",
		Data: TestData{UnitValue: 200, Quantity: 1},
	}
	if err := calc.AddNode(newNode, "1"); err != nil {
		t.Fatalf("AddNode() error = %v", err)
	}

	// 重新计算（版本号已递增，重新加载数据）
	result2, err := calc.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	if result2.TotalBase != 200 {
		t.Errorf("TotalBase after add = %d, want 200", result2.TotalBase)
	}

	// 更新节点
	if err := calc.UpdateNode("2", func(n *tree.Node[TestData]) {
		n.Data.UnitValue = 300
	}); err != nil {
		t.Fatalf("UpdateNode() error = %v", err)
	}

	result3, err := calc.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	if result3.TotalBase != 300 {
		t.Errorf("TotalBase after update = %d, want 300", result3.TotalBase)
	}

	// 删除节点
	if err := calc.RemoveNode("2"); err != nil {
		t.Fatalf("RemoveNode() error = %v", err)
	}

	result4, err := calc.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	if result4.TotalBase != 100 {
		t.Errorf("TotalBase after remove = %d, want 100", result4.TotalBase)
	}

	// 删除不存在的节点
	if err := calc.RemoveNode("999"); err == nil {
		t.Error("RemoveNode() should error for non-existent node")
	}
}

func TestCalculator_ItemCRUD(t *testing.T) {
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 100, Quantity: 1}},
		},
		items: []allocate.Item{
			{ID: "fee1", Name: "费用1", Value: 100},
		},
	}

	calc := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:item", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
	)

	result1, _ := calc.Calculate()
	if result1.TotalAlloc != 100 {
		t.Errorf("TotalAlloc = %d, want 100", result1.TotalAlloc)
	}

	// 添加分摊项
	if err := calc.AddItem(allocate.Item{ID: "fee2", Name: "费用2", Value: 200}); err != nil {
		t.Fatalf("AddItem() error = %v", err)
	}

	result2, _ := calc.Calculate()
	if result2.TotalAlloc != 300 {
		t.Errorf("TotalAlloc after add = %d, want 300", result2.TotalAlloc)
	}

	// 更新分摊项
	if err := calc.UpdateItem("fee2", func(item *allocate.Item) {
		item.Value = 500
	}); err != nil {
		t.Fatalf("UpdateItem() error = %v", err)
	}

	result3, _ := calc.Calculate()
	if result3.TotalAlloc != 600 {
		t.Errorf("TotalAlloc after update = %d, want 600", result3.TotalAlloc)
	}

	// 删除分摊项
	if err := calc.RemoveItem("fee2"); err != nil {
		t.Fatalf("RemoveItem() error = %v", err)
	}

	result4, _ := calc.Calculate()
	if result4.TotalAlloc != 100 {
		t.Errorf("TotalAlloc after remove = %d, want 100", result4.TotalAlloc)
	}
}

func TestCalculator_ExtraCRUD(t *testing.T) {
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 100, Quantity: 1}},
		},
		extraRate:  0.13,
		extraRatio: 0.13,
	}

	calc := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:extra", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
	)

	// 第一次计算（有加成）
	result1, _ := calc.Calculate()
	if result1.TotalExtra == 0 {
		t.Error("TotalExtra should not be 0")
	}

	// 移除加成
	if err := calc.RemoveExtra(); err != nil {
		t.Fatalf("RemoveExtra() error = %v", err)
	}

	result2, _ := calc.Calculate()
	if result2.TotalExtra != 0 {
		t.Errorf("TotalExtra after remove = %d, want 0", result2.TotalExtra)
	}

	// 重新设置加成
	if err := calc.SetExtra(0.20, 0.0); err != nil {
		t.Fatalf("SetExtra() error = %v", err)
	}

	result3, _ := calc.Calculate()
	if result3.TotalExtra != 25 {
		t.Errorf("TotalExtra after set = %d, want 25", result3.TotalExtra)
	}
}

func TestCalculator_Recalculate(t *testing.T) {
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 100, Quantity: 1}},
		},
	}

	cache := allocate.NewMemoryCache()

	calc := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:recalc", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)

	result1, _ := calc.Calculate()
	total1 := result1.TotalBase

	result2, _ := calc.Recalculate()
	if result2.TotalBase != total1 {
		t.Errorf("Recalculate result different: %d vs %d", result2.TotalBase, total1)
	}
}

// TestCalculator_MultiUser 测试多用户共享场景
// 用户A 修改数据后，用户B 通过更新 bizVersion 看到新数据
func TestCalculator_MultiUser(t *testing.T) {
	// 模拟共享数据存储
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 100, Quantity: 1}},
		},
	}

	cache := allocate.NewMemoryCache()

	// 用户A 创建 Calculator
	calcA := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("project:123", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)

	resultA1, _ := calcA.Calculate()
	if resultA1.TotalBase != 100 {
		t.Errorf("User A TotalBase = %d, want 100", resultA1.TotalBase)
	}

	// 用户B 创建 Calculator（相同 bizID+version，命中缓存）
	calcB := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("project:123", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)

	resultB1, _ := calcB.Calculate()
	if resultB1.TotalBase != 100 {
		t.Errorf("User B TotalBase = %d, want 100", resultB1.TotalBase)
	}

	// 用户A 修改数据（修改共享存储，并更新自己的 Calculator）
	store.nodes = append(store.nodes, tree.Node[TestData]{
		ID: "2", Pid: "1", Path: "/1/2/", Ppath: "/1/",
		Data: TestData{UnitValue: 200, Quantity: 1},
	})

	// 用户A 更新自己的数据（版本号自动递增到2）
	if err := calcA.AddNode(tree.Node[TestData]{
		ID: "2", Pid: "1", Path: "/1/2/", Ppath: "/1/",
		Data: TestData{UnitValue: 200, Quantity: 1},
	}, "1"); err != nil {
		t.Fatalf("AddNode() error = %v", err)
	}

	resultA2, _ := calcA.Calculate()
	if resultA2.TotalBase != 200 {
		t.Errorf("User A after add TotalBase = %d, want 200", resultA2.TotalBase)
	}

	// 用户B 使用旧版本号，仍然看到旧缓存
	resultB2, _ := calcB.Calculate()
	if resultB2.TotalBase != 100 {
		t.Errorf("User B with old version TotalBase = %d, want 100 (old cache)", resultB2.TotalBase)
	}

	// 用户B 更新版本号，看到新数据
	calcB2 := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("project:123", 2), // 新版本号
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)
	resultB3, _ := calcB2.Calculate()
	if resultB3.TotalBase != 200 {
		t.Errorf("User B with new version TotalBase = %d, want 200", resultB3.TotalBase)
	}
}

// TestCalculator_Idempotency 测试幂等性：相同 bizID+version 返回相同结果
func TestCalculator_Idempotency(t *testing.T) {
	store := &mockDataStore{
		nodes: []tree.Node[TestData]{
			{ID: "1", Pid: "", Path: "/1/", Ppath: "/", Data: TestData{UnitValue: 100, Quantity: 1}},
		},
	}

	cache := allocate.NewMemoryCache()

	// 第一次创建 Calculator 并计算
	calc1 := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:123", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)
	result1, err := calc1.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	// 第二次创建新的 Calculator，使用相同 bizID+version
	calc2 := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:123", 1),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)
	result2, err := calc2.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	// 结果应该相同（命中缓存）
	if result1.TotalBase != result2.TotalBase {
		t.Errorf("Idempotency failed: %d vs %d", result1.TotalBase, result2.TotalBase)
	}

	// 版本号不同，重新计算
	calc3 := allocate.NewCalculator(
		allocate.WithCacheKey[TestData]("test:123", 2),
		allocate.WithDataLoader[TestData](store),
		allocate.WithWeightFn(func(n *tree.Node[TestData]) float64 {
			return float64(n.Data.UnitValue) * n.Data.Quantity
		}),
		allocate.WithCache[TestData](cache),
	)
	result3, err := calc3.Calculate()
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	// 重新计算，结果应该相同（因为输入相同）
	if result1.TotalBase != result3.TotalBase {
		t.Errorf("Should recalculate for new version: %d vs %d", result1.TotalBase, result3.TotalBase)
	}
}
