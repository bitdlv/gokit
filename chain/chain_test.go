package chain

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMakeChain(t *testing.T) {
	handlers := []Handler[string, string]{
		func(c *Chain[string, string]) {
			c.AppendResult("1")
		},
		func(c *Chain[string, string]) {
			c.Next()
			c.AppendResult("3")
		},
		func(c *Chain[string, string]) {
			c.AppendResult("2")
			c.AppendResult("test")
		},
		func(c *Chain[string, string]) {
			c.Abort()
		},
		func(c *Chain[string, string]) {
			c.AppendResult("test2") //上面终止，不会执行到这个地方
		},
	}

	c := New(handlers)
	c.Exec("")
	results := c.Results()
	require.Equal(t, 4, len(results))
	require.Equal(t, "1", results[0])
	require.Equal(t, "2", results[1])
	require.Equal(t, "test", results[2])
	require.Equal(t, "3", results[3])
}
