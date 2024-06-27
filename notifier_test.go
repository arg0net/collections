package collections_test

import (
	"context"
	"math/rand"
	"testing"
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
