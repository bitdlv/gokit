package helper

import "sync"

func NewRepeatChecker[Group comparable, T comparable](cap int) *RepeatChecker[Group, T] {
	m := &RepeatChecker[Group, T]{
		m: make(map[Group]map[T]struct{}),
	}
	return m
}

type RepeatChecker[Group comparable, T comparable] struct {
	m    map[Group]map[T]struct{}
	lock sync.Mutex
}

func (r *RepeatChecker[Group, T]) Check(item T, group ...Group) bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	var g Group
	if len(group) > 0 {
		g = group[0]
	}
	var (
		gp map[T]struct{}
		ok bool
	)
	if gp, ok = r.m[g]; !ok {
		gp = make(map[T]struct{}, 100)
		r.m[g] = gp
	}
	if _, ok = gp[item]; ok {
		return false
	} else {
		gp[item] = struct{}{}
		return true
	}
}

func (r *RepeatChecker[Group, T]) GetItems() (result []T) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for _, m := range r.m {
		for k := range m {
			result = append(result, k)
		}
	}

	return
}
