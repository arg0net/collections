package collections_test

import (
	"context"
	"math/rand"
	"testing"
	"sync"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arg0net/collections"
)

func TestNotifier(t *testing.T) {
	sn := collections.NewStatefulNotifier(0)
	sn.Store(1)
	sn.Store(2)
	sn.Store(2)
	sn.Store(3)

	v, ch := sn.Load()
	require.Equal(t, 3, v)
	require.NotNil(t, ch)

	sn.Store(4)
	<-ch

	v, _ = sn.Load()
	require.Equal(t, 4, v)
}

func TestNotifierUpdate(t *testing.T) {
	sn := collections.NewStatefulNotifier(0)
	start := make(chan struct{})

	incr := func(in int) int {
		return in+1
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-start:
			}
			sn.Update(incr)
		}()
	}
	close(start)

	wg.Wait()
	v, _ := sn.Load()
	require.Equal(t, 10, v)
}

func TestNotifierWait(t *testing.T) {
	ctx := context.Background()

	done := make(chan int, 1)
	sn := collections.NewStatefulNotifier(0)
	go func() {
		v, _ := sn.Wait(ctx, func(v int) bool {
			return v == 3
		})
		done <- v
	}()

	// give time for wait to start.
	time.Sleep(10 * time.Millisecond)
	sn.Store(1)
	require.Empty(t, done)
	sn.Store(2)
	require.Empty(t, done)
	sn.Store(3)

	v := <-done
	require.Equal(t, 3, v)
}

func TestWaitCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	result := make(chan error, 1)
	sn := collections.NewStatefulNotifier(0)
	go func() {
		_, err := sn.Wait(ctx, func(v int) bool {
			return v == 42
		})
		result <- err
	}()

	// give time for wait to start.
	time.Sleep(10 * time.Millisecond)
	cancel()

	err := <-result
	require.ErrorIs(t, err, context.Canceled)
}

func TestNotifierWaitAny(t *testing.T) {
	ctx := context.Background()

	done := make(chan int, 1)
	sn := make([]*collections.StatefulNotifier[int], 5)
	for i := range sn {
		sn[i] = collections.NewStatefulNotifier(0)
	}
	go func() {
		_, idx := collections.WaitAny(ctx, func(v int) bool {
			return v == 42
		}, sn...)
		done <- idx
	}()

	// give time for wait to start.
	time.Sleep(10 * time.Millisecond)
	expected := rand.Intn(4)
	sn[0].Store(0)
	sn[expected].Store(42)

	got := <-done
	require.Equal(t, expected, got)
}

func TestNotifierWaitAnyCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	result := make(chan int, 1)
	sn := make([]*collections.StatefulNotifier[int], 5)
	for i := range sn {
		sn[i] = collections.NewStatefulNotifier(0)
	}
	go func() {
		_, idx := collections.WaitAny(ctx, func(v int) bool {
			return v == 42
		}, sn...)
		result <- idx
	}()

	// give time for wait to start.
	time.Sleep(10 * time.Millisecond)
	cancel()

	idx := <-result
	require.Equal(t, -1, idx)
}

func TestNotifierWaitAnyImmediate(t *testing.T) {
	ctx := context.Background()

	sn := make([]*collections.StatefulNotifier[int], 5)
	for i := range sn {
		sn[i] = collections.NewStatefulNotifier(i)
	}

	got, idx := collections.WaitAny(ctx, func(v int) bool {
		return v == 1
	}, sn...)
	require.Equal(t, 1, idx)
	require.Equal(t, 1, got)
}
