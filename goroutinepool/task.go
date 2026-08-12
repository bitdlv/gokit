package goroutinepool

func NewTask(params ...any) *Task {
	return NewTaskWithHandler(nil, params...)
}

func NewTaskWithHandler(h Handler, params ...any) *Task {
	return &Task{
		handler: h,
		params:  params,
	}
}

type Task struct {
	handler Handler
	params  []any
}
