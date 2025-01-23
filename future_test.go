package collections_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/arg0net/collections"
	"github.com/stretchr/testify/require"
)

func TestFuture(t *testing.T) {
	f := collections.NewFuture[int]()
	f.Set(1)
	value, err := f.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, value)
}

func TestFuture_SetTwice(t *testing.T) {
	f := collections.NewFuture[int]()
	f.Set(1)
	require.False(t, f.Set(2))
}

func TestFuture_GetCancelled(t *testing.T) {
	f := collections.NewFuture[int]()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := f.Get(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestWaitFutures(t *testing.T) {
	f1 := collections.NewFuture[int]()
	f2 := collections.NewFuture[int]()
	f3 := collections.NewFuture[int]()

	go func() {
		time.Sleep(10 * time.Millisecond)
		f1.Set(1)
	}()
	go func() {
		time.Sleep(30 * time.Millisecond)
		f2.Set(2)
	}()
	go func() {
		time.Sleep(20 * time.Millisecond)
		f3.Set(3)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results := []int{}
	it := collections.WatchFutures(ctx, f1, f2, f3)
	for _, v := range it {
		results = append(results, v)
	}
	// results will be out of order.
	require.NotEqual(t, []int{1, 2, 3}, results)
	slices.Sort(results)
	require.Equal(t, []int{1, 2, 3}, results)
}
