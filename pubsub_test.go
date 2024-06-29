package collections_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arg0net/collections"
)

func TestPubSub(t *testing.T) {
	var c collections.Channel[int]

	// Subscribe to the channel.
	seen := make([]atomic.Bool, 64)
	sub := c.Subscribe(func(index int) {
		seen[index].Store(true)
	})
	defer sub.Cancel()

	// Publish a value to the channel.
	for _, i := range rand.Perm(64) {
		go func() {
			c.Publish(i)
		}()
	}

	require.Eventually(t, func() bool {
		for i := range seen {
			if !seen[i].Load() {
				t.Logf("missing value: %d", i)
				return false
			}
		}
		return true
	}, 2*time.Second, 10*time.Millisecond)
}

func TestPubSub_Watch(t *testing.T) {
	var c collections.Channel[int]

	ctx, cancel := context.WithCancel(context.Background())
	received := make(chan int, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.Watch(ctx, func(index int) error {
			received <- index
			return nil
		})
	}()

	// Downside of Watch is that there's no way to be certain that it has been
	// setup. So we need to wait a bit before publishing.
	time.Sleep(100 * time.Millisecond)
	c.Publish(42)
	select {
	case <-time.After(2 * time.Second):
		require.Fail(t, "timeout")
	case got := <-received:
		require.Equal(t, 42, got)
	}

	require.Empty(t, done)
	cancel()
	err := <-done
	require.Error(t, err)
}

func BenchmarkPubSub(b *testing.B) {
	for _, n := range []int{0, 1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("PubSub-%d", n), func(b *testing.B) {
			benchmarkPubSubN(b, n)
		})
	}
}

func benchmarkPubSubN(b *testing.B, n int) {
	var c collections.Channel[int]

	// Setup n subscribers.
	received := make([]*atomic.Int64, n)
	subs := make([]*collections.Subscription[int], n)
	for i := 0; i < n; i++ {
		received[i] = new(atomic.Int64)
		subs[i] = c.Subscribe(func(_ int) {
			received[i].Add(1)
		})
		defer subs[i].Cancel()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Publish(i)
	}

	// Wait for all values to be received.
	require.Eventually(b, func() bool {
		for _, v := range received {
			if v.Load() != int64(b.N) {
				return false
			}
		}
		return true
	}, 2*time.Second, 1*time.Millisecond)
}
