package chain

type Handler[T any, M any] func(c *Chain[T, M])

func New[T any, M any](handlers []Handler[T, M]) *Chain[T, M] {
	return &Chain[T, M]{
		handlers: handlers,
		index:    -1,
		results:  make([]M, 0, len(handlers)),
	}
}

type Chain[T any, M any] struct {
	handlers []Handler[T, M]
	index    int8
	Data     T
	results  []M
	abort    bool
}

func (c *Chain[T, M]) Next() {
	if c.abort {
		return
	}

	c.index++
	for c.index < int8(len(c.handlers)) {
		c.handlers[c.index](c)
		if c.abort {
			return
		}
		c.index++
	}
}

func (c *Chain[T, M]) Exec(data T) {
	c.Data = data
	c.Next()
	defer func() {
		c.index = -1
	}()
}

func (c *Chain[T, M]) AppendResult(results ...M) {
	c.results = append(c.results, results...)
}

func (c *Chain[T, M]) Abort() {
	c.abort = true
}

func (c *Chain[T, M]) Results() []M {
	defer func() {
		c.results = make([]M, 0, len(c.handlers))
	}()
	return c.results
}
