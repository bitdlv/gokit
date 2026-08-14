package allocate

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/bitdlv/gokit/tree"
)

// nodeState 内部计算状态
type nodeState[T any] struct {
	node *tree.Node[T]

	// 权重与基础值
	weight       float64 // 权重（用于分摊比例）
	baseValue    int64   // 基础值（权重转换后的实际值）
	baseValueNet int64   // 净值基础值
	realValue    int64   // 实际业务值（如金额）

	// 分摊结果
	allocs map[string]*AllocDetail

	// 汇总
	subTotal    int64
	subTotalNet int64
	extra       int64
	total       int64
	totalNet    int64

	children []*nodeState[T]
}

// Calculator 通用分摊计算器
// 注意：Calculator 实例不是并发安全的，不要并发调用其方法
//
// 核心设计：
//   - 通过 WithCacheKey(bizID, bizVersion) 标识业务数据版本
//   - 通过 WithDataLoader 设置数据加载器，Calculate() 时自动加载
//   - 相同 bizID+bizVersion 的实例共享缓存
//   - 数据变更后调用方更新 bizVersion，自动加载新数据
type Calculator[T any] struct {
	// 业务标识
	bizID      string // 业务ID（如项目ID、订单ID）
	bizVersion int64  // 业务版本号

	// 数据加载器（必须设置，用于自动加载节点数据）
	dataLoader DataLoader[T]

	// 配置（从 dataLoader 加载后填充）
	nodes      []tree.Node[T]
	items      []Item
	extraRate  float64
	extraRatio float64

	// 用户自定义函数
	weightFn func(*tree.Node[T]) float64
	ratioFn  func(*tree.Node[T]) float64

	// 内部状态
	roots    []*nodeState[T]
	nodeByID map[string]*nodeState[T]

	// 缓存
	cache    Cache
	cacheTTL time.Duration

	// 加载状态
	loaded bool // 是否已从 dataLoader 加载数据
}

// NewCalculator 创建通用分摊计算器
// 必须设置 WithCacheKey 和 WithDataLoader
func NewCalculator[T any](opts ...Option[T]) *Calculator[T] {
	c := &Calculator[T]{
		nodeByID: make(map[string]*nodeState[T]),
		cacheTTL: 2 * time.Hour,
	}

	for _, opt := range opts {
		opt(c)
	}

	// 默认权重函数：如果用户未设置，使用默认值 1
	if c.weightFn == nil {
		c.weightFn = func(n *tree.Node[T]) float64 { return 1.0 }
	}
	if c.ratioFn == nil {
		c.ratioFn = func(n *tree.Node[T]) float64 { return 0 }
	}

	// 默认缓存
	if c.cache == nil {
		c.cache = NewMemoryCache()
	}

	return c
}

// Calculate 执行完整计算
// 自动从 dataLoader 加载 bizID+bizVersion 对应的数据
// 相同 bizID+bizVersion 的实例命中同一缓存
func (c *Calculator[T]) Calculate() (*Result, error) {
	// 自动加载数据（如果尚未加载）
	if err := c.ensureLoaded(); err != nil {
		return nil, fmt.Errorf("load data: %w", err)
	}

	// 检查缓存
	if c.cache != nil && c.bizID != "" {
		cacheKey := c.getCacheKey()
		if cached, err := c.cache.Get(context.Background(), cacheKey); err == nil && cached != nil {
			return cached, nil
		}
	}

	// 执行计算
	if err := c.buildTree(); err != nil {
		return nil, fmt.Errorf("build tree: %w", err)
	}
	c.calcBaseValues()
	c.distributeItems()
	c.calcSummary()

	// 构建结果
	result := c.buildResult()

	// 写入缓存
	if c.cache != nil && c.bizID != "" {
		cacheKey := c.getCacheKey()
		_ = c.cache.Set(context.Background(), cacheKey, result, c.cacheTTL)
	}

	return result, nil
}

// ensureLoaded 确保数据已从 dataLoader 加载
func (c *Calculator[T]) ensureLoaded() error {
	if c.loaded {
		return nil
	}
	if c.dataLoader == nil {
		return fmt.Errorf("dataLoader is required, please set WithDataLoader")
	}
	if c.bizID == "" {
		return fmt.Errorf("bizID is required, please set WithCacheKey")
	}

	ctx := context.Background()
	nodes, items, extraRate, extraRatio, err := c.dataLoader.Load(ctx, c.bizID, c.bizVersion)
	if err != nil {
		return fmt.Errorf("load data for %s:v%d: %w", c.bizID, c.bizVersion, err)
	}

	c.nodes = nodes
	c.items = items
	c.extraRate = extraRate
	c.extraRatio = extraRatio
	c.loaded = true

	return nil
}

// Invalidate 使缓存失效
func (c *Calculator[T]) Invalidate() error {
	if c.cache != nil && c.bizID != "" {
		return c.cache.Delete(context.Background(), c.getCacheKey())
	}
	return nil
}

// getCacheKey 获取缓存键
// 格式: allocate:{bizID}:v{bizVersion}
// 示例: allocate:project:123:v5
func (c *Calculator[T]) getCacheKey() string {
	return fmt.Sprintf("allocate:%s:v%d", c.bizID, c.bizVersion)
}

// bumpVersion 递增业务版本号（数据变更时调用）
// 注意：递增后需要重新调用 Calculate()，会自动加载新版本数据
func (c *Calculator[T]) bumpVersion() {
	c.bizVersion++
	// 不设置 loaded=false，因为 CRUD 操作已经直接修改了本地数据
	// 只有 Recalculate() 才需要重新从 dataLoader 加载
}

// Reload 强制重新加载数据（从 dataLoader）
// 适用于数据已变更但 bizVersion 未更新的场景
func (c *Calculator[T]) Reload() error {
	c.loaded = false
	return c.Invalidate()
}

// ─────────────────────────────────────────────
// 树节点操作（添加/删除/修改）
// ─────────────────────────────────────────────

// AddNode 添加节点到树中
// parentID: 父节点ID，空字符串表示根节点
// 添加后自动失效缓存并递增版本号
func (c *Calculator[T]) AddNode(node tree.Node[T], parentID string) error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	// 设置父节点信息
	if parentID == "" {
		node.Pid = ""
		node.Ppath = "/"
	} else {
		parent, exists := c.nodeByID[parentID]
		if !exists {
			return fmt.Errorf("parent node %s not found", parentID)
		}
		node.Pid = parentID
		node.Ppath = parent.node.Path
	}

	// 添加到节点列表
	c.nodes = append(c.nodes, node)

	// 失效缓存并递增版本号
	c.bumpVersion()
	return c.Invalidate()
}

// AddNodes 批量添加节点
func (c *Calculator[T]) AddNodes(nodes []tree.Node[T]) error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	for _, node := range nodes {
		parentID := node.Pid
		if parentID == "" {
			node.Ppath = "/"
		} else {
			parent, exists := c.nodeByID[parentID]
			if !exists {
				return fmt.Errorf("parent node %s not found for node %s", parentID, node.ID)
			}
			node.Ppath = parent.node.Path
		}
		c.nodes = append(c.nodes, node)
	}
	c.bumpVersion()
	return c.Invalidate()
}

// RemoveNode 从树中删除节点（包括其子树）
// 删除后自动失效缓存并递增版本号
func (c *Calculator[T]) RemoveNode(nodeID string) error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	// 找到要删除的节点
	target, exists := c.nodeByID[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// 收集要删除的所有节点ID（包括子树）
	toDelete := make(map[string]bool)
	var collect func(*nodeState[T])
	collect = func(ns *nodeState[T]) {
		toDelete[ns.node.ID] = true
		for _, child := range ns.children {
			collect(child)
		}
	}
	collect(target)

	// 从 nodes 中移除
	newNodes := make([]tree.Node[T], 0, len(c.nodes))
	for _, n := range c.nodes {
		if !toDelete[n.ID] {
			newNodes = append(newNodes, n)
		}
	}
	c.nodes = newNodes

	// 失效缓存并递增版本号
	c.bumpVersion()
	return c.Invalidate()
}

// UpdateNode 更新节点数据
// 更新后自动失效缓存并递增版本号
func (c *Calculator[T]) UpdateNode(nodeID string, updateFn func(*tree.Node[T])) error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	// 找到节点
	ns, exists := c.nodeByID[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// 更新数据
	updateFn(ns.node)

	// 同步更新 nodes 列表
	for i, n := range c.nodes {
		if n.ID == nodeID {
			c.nodes[i] = *ns.node
			break
		}
	}

	// 失效缓存并递增版本号
	c.bumpVersion()
	return c.Invalidate()
}

// ─────────────────────────────────────────────
// 分摊项操作（添加/删除/修改）
// ─────────────────────────────────────────────

// AddItem 添加分摊项
// 添加后自动失效缓存并递增版本号
func (c *Calculator[T]) AddItem(item Item) error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	// 检查是否已存在
	for _, existing := range c.items {
		if existing.ID == item.ID {
			return fmt.Errorf("item %s already exists", item.ID)
		}
	}

	c.items = append(c.items, item)
	c.bumpVersion()
	return c.Invalidate()
}

// AddItems 批量添加分摊项
func (c *Calculator[T]) AddItems(items []Item) error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	for _, item := range items {
		for _, existing := range c.items {
			if existing.ID == item.ID {
				return fmt.Errorf("item %s already exists", item.ID)
			}
		}
		c.items = append(c.items, item)
	}
	c.bumpVersion()
	return c.Invalidate()
}

// RemoveItem 删除分摊项
// 删除后自动失效缓存并递增版本号
func (c *Calculator[T]) RemoveItem(itemID string) error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	newItems := make([]Item, 0, len(c.items))
	found := false
	for _, item := range c.items {
		if item.ID == itemID {
			found = true
			continue
		}
		newItems = append(newItems, item)
	}

	if !found {
		return fmt.Errorf("item %s not found", itemID)
	}

	c.items = newItems
	c.bumpVersion()
	return c.Invalidate()
}

// UpdateItem 更新分摊项
// 更新后自动失效缓存并递增版本号
func (c *Calculator[T]) UpdateItem(itemID string, updateFn func(*Item)) error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	for i, item := range c.items {
		if item.ID == itemID {
			updateFn(&c.items[i])
			c.bumpVersion()
			return c.Invalidate()
		}
	}
	return fmt.Errorf("item %s not found", itemID)
}

// ─────────────────────────────────────────────
// 加成比例操作（添加/删除/修改）
// ─────────────────────────────────────────────

// SetExtra 设置加成比例和比例因子
// 设置后自动失效缓存并递增版本号
func (c *Calculator[T]) SetExtra(rate, ratio float64) error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	c.extraRate = rate
	c.extraRatio = ratio
	c.bumpVersion()
	return c.Invalidate()
}

// RemoveExtra 移除加成（设置为0）
// 移除后自动失效缓存并递增版本号
func (c *Calculator[T]) RemoveExtra() error {
	if err := c.ensureLoaded(); err != nil {
		return err
	}

	c.extraRate = 0
	c.extraRatio = 0
	c.bumpVersion()
	return c.Invalidate()
}

// ─────────────────────────────────────────────
// 重新计算（强制刷新）
// ─────────────────────────────────────────────

// Recalculate 强制重新计算（忽略缓存，重新从 dataLoader 加载）
func (c *Calculator[T]) Recalculate() (*Result, error) {
	c.loaded = false
	if err := c.Invalidate(); err != nil {
		return nil, err
	}
	return c.Calculate()
}

// buildTree 构建树结构
func (c *Calculator[T]) buildTree() error {
	if len(c.nodes) == 0 {
		return fmt.Errorf("nodes is empty")
	}

	// 重置内部状态（支持重复调用）
	c.roots = nil
	c.nodeByID = make(map[string]*nodeState[T])

	// 使用 tree.BuildByPath 构建树
	roots := tree.BuildByPath(c.nodes)

	// 转换为内部状态节点
	var convert func(*tree.Node[T]) *nodeState[T]
	convert = func(n *tree.Node[T]) *nodeState[T] {
		ns := &nodeState[T]{
			node:   n,
			allocs: make(map[string]*AllocDetail),
		}
		for _, child := range n.Child {
			ns.children = append(ns.children, convert(child))
		}
		c.nodeByID[n.ID] = ns
		return ns
	}

	for _, r := range roots {
		c.roots = append(c.roots, convert(r))
	}

	return nil
}

// calcBaseValues 计算各节点基础值（后序遍历）
func (c *Calculator[T]) calcBaseValues() {
	var dfs func(*nodeState[T])
	dfs = func(ns *nodeState[T]) {
		n := ns.node

		if len(n.Child) == 0 {
			// 叶子节点：使用用户定义的权重函数
			ns.weight = c.weightFn(n)
		} else {
			// 中间节点：先计算子节点
			var childWeightSum float64
			for _, child := range ns.children {
				dfs(child)
				childWeightSum += child.weight
			}

			// 中间节点权重 = 子节点权重和 * max(quantity, 1)
			// 从节点数据中提取 quantity（尝试常见字段）
			qty := extractQuantity(n.Data)
			if qty <= 0 {
				qty = 1
			}
			ns.weight = childWeightSum * qty
		}

		// 权重转基础值
		ns.baseValue = int64(math.Round(ns.weight))
		ns.realValue = ns.baseValue // 默认实际值=基础值，可通过 valueFn 自定义

		// 计算净值（按 ratio 拆分）
		ratio := c.ratioFn(n)
		if ratio > 0 {
			ns.baseValueNet = int64(math.Round(float64(ns.baseValue) / (1 + ratio)))
		} else {
			ns.baseValueNet = ns.baseValue
		}
	}

	for _, r := range c.roots {
		dfs(r)
	}
}

// distributeItems 分摊所有费用项
func (c *Calculator[T]) distributeItems() {
	if len(c.items) == 0 || len(c.roots) == 0 {
		return
	}

	// 计算总权重（用于根级分摊）
	var totalWeight float64
	for _, r := range c.roots {
		totalWeight += r.weight
	}
	if totalWeight <= 0 {
		return
	}

	// 对每个费用项进行分摊
	for _, item := range c.items {
		if item.Value <= 0 {
			continue
		}

		// 计算净值
		var valueNet int64
		if item.Ratio > 0 {
			valueNet = int64(math.Round(float64(item.Value) / (1 + item.Ratio)))
		} else {
			valueNet = item.Value
		}

		// 根级分摊
		type rootSlot struct {
			ns    *nodeState[T]
			v, vn int64
		}
		slots := make([]*rootSlot, len(c.roots))
		for i, r := range c.roots {
			slots[i] = &rootSlot{ns: r}
		}

		// 使用最大余数法分摊
		AllocateAtLeastOne(item.Value, slots, func(s *rootSlot) float64 { return s.ns.weight }, func(s *rootSlot, v int64) { s.v = v })
		if valueNet > 0 {
			AllocateAtLeastOne(valueNet, slots, func(s *rootSlot) float64 { return s.ns.weight }, func(s *rootSlot, v int64) { s.vn = v })
		}

		// 递归分配到子树
		for _, s := range slots {
			c.distributeToNode(s.ns, s.v, s.vn, item)
		}
	}
}

// distributeToNode 递归分摊到节点及其子树
func (c *Calculator[T]) distributeToNode(ns *nodeState[T], value, valueNet int64, item Item) {
	if value <= 0 && valueNet <= 0 {
		return
	}

	// 记录本节点分摊
	alloc, ok := ns.allocs[item.ID]
	if !ok {
		alloc = &AllocDetail{
			ItemID:   item.ID,
			ItemName: item.Name,
			Ratio:    item.Ratio,
		}
		ns.allocs[item.ID] = alloc
	}
	alloc.Value += value
	alloc.ValueNet += valueNet

	// 叶子节点，分配完毕
	if len(ns.children) == 0 {
		return
	}

	// 按子树权重递归分配
	type childSlot struct {
		ns    *nodeState[T]
		v, vn int64
	}
	slots := make([]*childSlot, len(ns.children))
	for i, child := range ns.children {
		slots[i] = &childSlot{ns: child}
	}

	if value > 0 {
		AllocateAtLeastOne(value, slots, func(s *childSlot) float64 { return s.ns.weight }, func(s *childSlot, v int64) { s.v = v })
	}
	if valueNet > 0 {
		AllocateAtLeastOne(valueNet, slots, func(s *childSlot) float64 { return s.ns.weight }, func(s *childSlot, v int64) { s.vn = v })
	}

	for _, s := range slots {
		c.distributeToNode(s.ns, s.v, s.vn, item)
	}
}

// calcSummary 汇总计算 + 加成
func (c *Calculator[T]) calcSummary() {
	var dfs func(*nodeState[T])
	dfs = func(ns *nodeState[T]) {
		// 先计算子节点
		for _, child := range ns.children {
			dfs(child)
		}

		// 汇总分摊
		var allocTotal, allocTotalNet int64
		for _, a := range ns.allocs {
			allocTotal += a.Value
			allocTotalNet += a.ValueNet
		}

		// 小计
		ns.subTotal = ns.realValue + allocTotal
		ns.subTotalNet = ns.baseValueNet + allocTotalNet

		// 加成计算（基于净值）
		if c.extraRate > 0 && c.extraRate < 1 {
			ns.extra = int64(math.Round(float64(ns.subTotalNet) * c.extraRate / (1 - c.extraRate)))
		}

		// 总量
		ns.totalNet = ns.subTotalNet + ns.extra
		if c.extraRatio > 0 {
			ns.total = int64(math.Round(float64(ns.totalNet) * (1 + c.extraRatio)))
		} else {
			ns.total = ns.totalNet
		}
	}

	for _, r := range c.roots {
		dfs(r)
	}
}

// buildResult 构建输出结果
func (c *Calculator[T]) buildResult() *Result {
	result := &Result{
		NodeIndex: make(map[string]*NodeDetail),
	}

	var convert func(*nodeState[T]) *NodeDetail
	convert = func(ns *nodeState[T]) *NodeDetail {
		nd := &NodeDetail{
			NodeID:       ns.node.ID,
			BaseWeight:   ns.weight,
			BaseValue:    ns.baseValue,
			BaseValueNet: ns.baseValueNet,
			SubTotal:     ns.subTotal,
			SubTotalNet:  ns.subTotalNet,
			Extra:        ns.extra,
			Total:        ns.total,
			TotalNet:     ns.totalNet,
		}

		// 转换分摊明细
		for _, a := range ns.allocs {
			nd.AllocList = append(nd.AllocList, a)
			nd.AllocTotal += a.Value
		}

		// 递归转换子节点
		for _, child := range ns.children {
			nd.Children = append(nd.Children, convert(child))
		}

		// 索引
		result.NodeIndex[nd.NodeID] = nd
		return nd
	}

	for _, r := range c.roots {
		nd := convert(r)
		result.Roots = append(result.Roots, nd)

		// 汇总
		result.TotalBase += nd.BaseValue
		result.TotalAlloc += nd.AllocTotal
		result.TotalExtra += nd.Extra
		result.GrandTotal += nd.Total
		result.GrandTotalNet += nd.TotalNet
	}

	return result
}

// extractQuantity 从数据中提取数量
// 支持以下类型：
//   - interface{ GetQuantity() float64 }
//   - interface{ GetQty() float64 }
//   - map[string]any 中的 "qty" 或 "quantity" 字段
func extractQuantity[T any](data T) float64 {
	switch v := any(data).(type) {
	case interface{ GetQuantity() float64 }:
		return v.GetQuantity()
	case interface{ GetQty() float64 }:
		return v.GetQty()
	case map[string]any:
		if q, ok := v["qty"].(float64); ok {
			return q
		}
		if q, ok := v["quantity"].(float64); ok {
			return q
		}
		if q, ok := v["qty"].(int64); ok {
			return float64(q)
		}
		if q, ok := v["quantity"].(int64); ok {
			return float64(q)
		}
	}
	return 1
}

// ─────────────────────────────────────────────
// 内存缓存实现
// ─────────────────────────────────────────────

// MemoryCache 内存缓存
type MemoryCache struct {
	mu   sync.RWMutex
	data map[string]*cacheEntry
}

type cacheEntry struct {
	result   *Result
	expireAt time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data: make(map[string]*cacheEntry),
	}
}

// Get 获取缓存
func (m *MemoryCache) Get(ctx context.Context, key string) (*Result, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.data[key]
	if !ok {
		return nil, nil
	}

	if time.Now().After(entry.expireAt) {
		return nil, nil
	}

	return entry.result, nil
}

// Set 设置缓存
func (m *MemoryCache) Set(ctx context.Context, key string, result *Result, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = &cacheEntry{
		result:   result,
		expireAt: time.Now().Add(ttl),
	}
	return nil
}

// Delete 删除缓存
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}
