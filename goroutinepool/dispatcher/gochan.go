package dispatcher

import (
	log "github.com/sirupsen/logrus"
)

type DispatchItem struct {
	TaskFunc TaskDispatchFunc
	ID       int
	Name     string
	Param    []byte
}

// TaskFunc task
type TaskDispatchFunc func(int, string, []byte) error

type gochan struct {
	uuid      int
	tasksChan chan DispatchItem
	dieChan   chan struct{}
}

// newGochan return gochan with bufferNum tasks
func newGochan(bufferNum int) *gochan {
	gc := &gochan{
		uuid:      defaultUUID(),
		tasksChan: make(chan DispatchItem, bufferNum),
		dieChan:   make(chan struct{}),
	}
	return gc
}

func (gc *gochan) setUUID(uuid int) {
	gc.uuid = uuid
}

// run make gochan running
func (gc *gochan) run() {
	go gc.start()
}

// start gochan's goroutine
func (gc *gochan) start() {
	defer func() {
		log.Infof("gochan %d ending...", gc.uuid)
	}()
	log.Infof("gochan %d starting...", gc.uuid)

	for {
		select {
		case task := <-gc.tasksChan:
			err := task.TaskFunc(task.ID, task.Name, task.Param)
			if err != nil {
				log.Errorf("task in gochan %d error: %s", gc.uuid, err.Error())
			}
		case <-gc.dieChan:
			return
		}
	}
}
