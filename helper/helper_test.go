package helper

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPluck(t *testing.T) {
	type test struct {
		ID   int32
		Name string
	}
	data := []test{
		{
			ID:   1,
			Name: "name1",
		},
		{
			ID:   2,
			Name: "name2",
		},
		{
			ID:   3,
			Name: "name3",
		},
	}

	type d struct {
		ID   []int32
		Name []string
	}
	c := d{}
	Pluck(data, &c)
	require.Equal(t, int32(1), c.ID[0])
	require.Equal(t, int32(2), c.ID[1])
	require.Equal(t, int32(3), c.ID[2])
	require.Equal(t, "name1", c.Name[0])
	require.Equal(t, "name2", c.Name[1])
	require.Equal(t, "name3", c.Name[2])
}

func TestGetField(t *testing.T) {
	type test struct {
		A struct {
			B *struct {
				C string
			}
		}
	}

	te := test{
		A: struct{ B *struct{ C string } }{B: &struct{ C string }{C: "123"}},
	}

	require.Equal(t, "123", GetField[string](te, "A.B.C"))
}

// func TestRepeatCheck(t *testing.T) {
// 	rc := NewRepeatChecker[string](5)
// 	require.True(t, rc.Check("1"))
// 	require.True(t, rc.Check("2"))
// 	require.False(t, rc.Check("2"))
// }

func TestTimeout(t *testing.T) {
	v := 0
	Timeout(func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			v++
		case <-ctx.Done():
		}
		return nil
	}, 4*time.Second)

	require.Equal(t, 0, v)

	Timeout(func(ctx context.Context) error {
		select {
		case <-time.After(4 * time.Second):
			v++
		case <-ctx.Done():
		}
		return nil
	}, 5*time.Second)

	require.Equal(t, 1, v)
}
