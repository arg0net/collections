package collections

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParallelConsume_ChannelGracefulShutdown(t *testing.T) {
	const numItems = 100
	const numWorkers = 10

	ch := &Channel[int]{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	recv := ch.Receive()
	// Start the producer
	go func() {
		time.Sleep(100 * time.Millisecond)
		for i := range numItems {
			ch.Publish(i)
		}
		ch.Close()
	}()

	var mu sync.Mutex
	var sum int
	var numWorking atomic.Int64
	var maxWorkers int

	err := Consume(ctx, numWorkers, recv, func(_ context.Context, i int) error {
		numWorking.Add(1)
		defer numWorking.Add(-1)
		dur := time.Duration(25+rand.Intn(25)) * time.Millisecond
		time.Sleep(dur)
		mu.Lock()
		if w := numWorking.Load(); w > int64(maxWorkers) {
			maxWorkers = int(w)
		}
		sum += i
		mu.Unlock()
		return nil
	})
	require.NoError(t, err)

	// Check all items were received, none lost
	require.Equal(t, numItems*(numItems-1)/2, sum)
	require.Equal(t, numWorkers, maxWorkers)
}
