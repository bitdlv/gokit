// Package goroutinepool 协程池
package goroutinepool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type Handler func(...any) error

func WithDefaultFunc(f Handler) func(*Pool) {
	return func(pool *Pool) {
		pool.defaultFunc = f
	}
}

// WithCap set cap. default 100.
func WithCap(cap int) func(*Pool) {
	if cap <= 0 {
		panic("cap can not less 0")
	}
	return func(pool *Pool) {
		pool.cap = cap
	}
}

func New(opts ...func(*Pool)) *Pool {
	p := &Pool{
		cap: 100,
	}
	for _, opt := range opts {
		opt(p)
	}
	p.taskChan = make(chan *Task, p.cap)
	return p
}

type Pool struct {
	defaultFunc Handler
	cap         int
	current     atomic.Uint64
	taskChan    chan *Task
	wg          sync.WaitGroup
	Err         error
	ctx         context.Context
	cancel      func()
}

func (p *Pool) Put(task *Task) {
	if p.ctx == nil {
		p.ctx, p.cancel = context.WithCancel(context.Background())
	}
	if uint64(p.cap) > p.current.Load() {
		p.RunWorker()
	}

	p.taskChan <- task
}

func (p *Pool) inc() {
	p.current.Add(1)
	p.wg.Add(1)
}

func (p *Pool) dec() {
	p.current.Add(^uint64(0))
	p.wg.Done()
}

func (p *Pool) RunWorker() {
	p.inc()

	go func() {
		defer p.dec()
		var err error
		for t := range p.taskChan {
			select {
			case <-p.ctx.Done():
				goto breaked
			default:
				if t.handler != nil {
					err = t.handler(t.params...)
					if err != nil {
						goto breaked
					}
					continue
				}

				if p.defaultFunc != nil {
					p.defaultFunc(t.params...)
					if err != nil {
						goto breaked
					}
					continue
				}

				err = errors.New("there is no handler found")
				goto breaked
			}
		}
	breaked:
		if err != nil && p.Err == nil {
			p.error(err)
		}
	}()
}

func (p *Pool) error(err error) {
	p.Err = err
	p.cancel()
}

func (p *Pool) Wait() {
	close(p.taskChan)
	p.wg.Wait()
}
