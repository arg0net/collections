package collections

import (
	"context"
	"iter"
	"reflect"
	"sync"
)

// Future is a value that will be set at some point in the future.
// It is similar to a StatefulNotifier, but can only be set once.
type Future[T any] struct {
	set   sync.Once
	value T
	done  chan struct{}
}

// NewFuture creates a new Future.
func NewFuture[T any]() *Future[T] {
	return &Future[T]{
		done: make(chan struct{}),
	}
}

// Done returns a channel that is unblocked when the Future has been set.
func (f *Future[T]) Done() <-chan struct{} {
	return f.done
}

// Get blocks until the value is available or the context is cancelled.
func (f *Future[T]) Get(ctx context.Context) (T, error) {
	select {
	case <-f.done:
		return f.value, nil
	case <-ctx.Done():
		return f.value, ctx.Err()
	}
}

// Set sets the value of the Future.
// This unblocks any calls to Get.
// It returns false if the Future has already been set.
func (f *Future[T]) Set(value T) bool {
	var wasSet bool
	f.set.Do(func() {
		f.value = value
		close(f.done)
		wasSet = true
	})
	return wasSet
}

// WatchFutures returns an iterator over the future results.
// It will yield the index and value of the futures as they are set,
// until the context is cancelled or all futures have been received.
func WatchFutures[T any](ctx context.Context, futures ...*Future[T]) iter.Seq2[int, T] {
	cases := make([]reflect.SelectCase, 0, len(futures)+1)
	for _, f := range futures {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(f.Done()),
		})
	}
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	})
	nilv := reflect.ValueOf(nil)

	return func(yield func(int, T) bool) {
		remaining := len(futures)
		for {
			chosen, _, _ := reflect.Select(cases)
			if chosen == len(futures) {
				return // context cancelled
			}
			future := futures[chosen]
			value, err := future.Get(ctx)
			if err != nil {
				return // context cancelled
			}
			if !yield(chosen, value) {
				return
			}
			cases[chosen].Chan = nilv // don't select this future again
			remaining--
			if remaining == 0 {
				return
			}
		}
	}
}
