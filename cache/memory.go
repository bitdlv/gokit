package cache

import (
	"sync"
	"time"
)

type wrapper struct {
	data   any
	expire int64 //过期时间
}

type MemoryCache[T any] struct {
	cache         map[any]*wrapper
	clearInterval time.Duration
	lock          sync.RWMutex
}

func NewMemoryCache[T any]() *MemoryCache[T] {
	m := &MemoryCache[T]{
		cache: make(map[any]*wrapper),
	}

	if m.clearInterval == 0 {
		m.clearInterval = time.Second * 5
	}

	go m.clean()

	return m
}

func (c *MemoryCache[T]) clean() {
	tick := time.NewTicker(c.clearInterval)
	defer tick.Stop()
	for range tick.C {
		c.lock.Lock()
		for k, v := range c.cache {
			if v.expire > 0 && v.expire < time.Now().Unix() {
				delete(c.cache, k)
			}
		}
		c.lock.Unlock()
	}
}

func (c *MemoryCache[T]) Get(key any) (data T, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if v, ok := c.cache[key]; ok {
		if v.expire > 0 && v.expire < time.Now().Unix() {
			delete(c.cache, key)
			return data, false
		}
		return v.data.(T), true
	}
	return data, false
}

func WithMemoryExpire(t time.Time) func(*wrapper) {
	return func(w *wrapper) {
		w.expire = t.Unix()
	}
}

func (c *MemoryCache[T]) Set(key any, data T, sets ...func(*wrapper)) {
	w := &wrapper{
		data: data,
	}

	for _, set := range sets {
		set(w)
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache[key] = w
}
