// Package scheduler 定时任务调度包,用法详见测试代码.
package scheduler

import (
	"context"
	"log"
	"sync"
	"time"
)

func NewScheduler() (s *Scheduler) {
	s = &Scheduler{
		jobs: make([]*Job, 0),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	go s.run()
	return
}

type Scheduler struct {
	jobs   []*Job
	lock   sync.Mutex
	ctx    context.Context
	cancel func()
	wait   sync.WaitGroup
}

func (s *Scheduler) Register(job *Job) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.jobs = append(s.jobs, job)
}

// RegisterAndRunImmediately 注册并立即执行一次
func (s *Scheduler) RegisterAndRunImmediately(job *Job) {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := job.Run()
	if err != nil {
		log.Printf("[scheduler] job error: %s", err.Error())
	}
	s.jobs = append(s.jobs, job)
}

func (s *Scheduler) MustRegisterAndRunImmediately(job *Job) {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := job.Run()
	if err != nil {
		log.Panicf("[scheduler] job error: %s", err.Error())
	}
	s.jobs = append(s.jobs, job)
}

func (s *Scheduler) run() {
	s.wait.Add(1)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			time.Sleep(50 * time.Millisecond)

			job := s.Pop()
			if job == nil {
				continue
			}
			err := job.Run()
			if err != nil {
				log.Printf("[scheduler] job error: %s", err.Error())
			}
			if !job.IsOnce() {
				s.Register(job)
			}
		}
	}
}

func (s *Scheduler) Pop() (job *Job) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var index int
	for index, job = range s.jobs {
		if job.IsTime() {
			s.jobs = append(s.jobs[:index], s.jobs[index+1:]...)
			return
		}
	}

	return nil
}

func (s *Scheduler) Close() {
	s.cancel()
	s.wait.Wait()
}
