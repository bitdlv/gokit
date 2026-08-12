package tree

import (
	"sync"
)

// Elem 结构体
type Elem struct {
	ID    string  `json:"id"`
	Pid   string  `json:"pid"`
	Name  string  `json:"name"`
	Child []*Elem `json:"child"`
}

// SafeListStore 并发安全容器
type SafeListStore struct {
	mu   sync.RWMutex
	data []Elem
}

func NewSafeListStore() *SafeListStore {
	return &SafeListStore{
		mu:   sync.RWMutex{},
		data: nil,
	}
}

// Add 添加元素（并发安全）
// 批量添加
func (s *SafeListStore) Add(items []Elem) {
	s.mu.Lock()
	s.data = append(s.data, items...)
	s.mu.Unlock()
}

// GetAll 获取全部元素（并发安全）
func (s *SafeListStore) GetAll() []Elem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// 返回副本，避免外部修改
	return append([]Elem(nil), s.data...)
}

// BuildTreeByCondition 按条件构建树
func (s *SafeListStore) BuildTreeByCondition(matchFunc func(l Elem) bool) *Elem {
	var matched *Elem
	var idMap = make(map[string]*Elem)
	var items = s.GetAll()
	for i := range items {
		idMap[items[i].ID] = &items[i]
	}
	for i := range items {
		if matchFunc(items[i]) {
			matched = &items[i]
			break
		}
	}
	if matched == nil {
		return nil
	}
	buildBranch(matched, idMap)
	return matched
}

// 递归调用
func buildBranch(node *Elem, idMap map[string]*Elem) {
	for _, l := range idMap {
		if l.Pid == node.ID {
			node.Child = append(node.Child, l)
			buildBranch(l, idMap)
		}
	}
}
