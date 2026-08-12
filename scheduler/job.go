package scheduler

import (
	"errors"
	"time"
)

type JobOption func(*Job)

func WithInterval(interval time.Duration) JobOption {
	return func(job *Job) {
		job.interval = interval
	}
}

func WithOnce(t time.Time) JobOption {
	return func(job *Job) {
		job.time = t
	}
}

func NewJob(f func() error, options ...JobOption) *Job {
	w := &Job{
		f:        f,
		lastTime: time.Now(),
	}

	for _, option := range options {
		option(w)
	}

	if w.time.IsZero() && w.interval.Seconds() < 1 {
		panic(errors.New("job is invalid"))
	}

	return w
}

type Job struct {
	interval time.Duration
	time     time.Time
	lastTime time.Time
	f        func() error
}

func (j *Job) Run() (err error) {
	j.lastTime = time.Now()
	return j.f()
}

func (j Job) IsOnce() bool {
	return !j.time.IsZero()
}

func (j Job) IsTime() bool {
	if j.IsOnce() {
		return time.Since(j.time) >= 0
	}

	now := time.Now()
	execTime := j.lastTime.Add(j.interval)
	return now.Sub(execTime) >= 0
}
