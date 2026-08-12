package paging_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/bitdlv/gokit/paging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchAll_MultiplePages_OrderedMerge(t *testing.T) {
	// 25 条数据, pageSize=10 → 3 页
	all := make([]int, 25)
	for i := range all {
		all[i] = i
	}
	total := int64(len(all))

	var calls int32
	fn := func(_ context.Context, page, pageSize int) (*paging.PageResult[int], error) {
		atomic.AddInt32(&calls, 1)
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > len(all) {
			end = len(all)
		}
		return &paging.PageResult[int]{Items: all[start:end], Total: total}, nil
	}

	got, err := paging.FetchAll(context.Background(), 10, 4, fn)
	require.NoError(t, err)
	assert.Equal(t, all, got)
	assert.EqualValues(t, 3, atomic.LoadInt32(&calls))
}

func TestFetchAll_FirstPageOnly(t *testing.T) {
	fn := func(_ context.Context, page, pageSize int) (*paging.PageResult[int], error) {
		return &paging.PageResult[int]{Items: []int{1, 2, 3}, Total: 3}, nil
	}
	got, err := paging.FetchAll(context.Background(), 10, 4, fn)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestFetchAll_EmptyTotal(t *testing.T) {
	fn := func(_ context.Context, page, pageSize int) (*paging.PageResult[int], error) {
		return &paging.PageResult[int]{Items: nil, Total: 0}, nil
	}
	got, err := paging.FetchAll(context.Background(), 10, 4, fn)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestFetchAll_ErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	fn := func(_ context.Context, page, pageSize int) (*paging.PageResult[int], error) {
		if page == 1 {
			return &paging.PageResult[int]{Items: []int{0}, Total: 100}, nil
		}
		if page == 3 {
			return nil, boom
		}
		items := make([]int, pageSize)
		return &paging.PageResult[int]{Items: items, Total: 100}, nil
	}
	_, err := paging.FetchAll(context.Background(), 10, 4, fn)
	assert.ErrorIs(t, err, boom)
}

func TestNormalize(t *testing.T) {
	p, s, o := paging.Normalize(0, 0)
	assert.Equal(t, 1, p)
	assert.Equal(t, 10, s)
	assert.Equal(t, 0, o)

	p, s, o = paging.Normalize(3, 20)
	assert.Equal(t, 3, p)
	assert.Equal(t, 20, s)
	assert.Equal(t, 40, o)

	p, s, o = paging.Normalize(0, 0, 50)
	assert.Equal(t, 1, p)
	assert.Equal(t, 50, s)
	assert.Equal(t, 0, o)
}
