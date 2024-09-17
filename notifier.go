package collections

import (
	"context"
	"sync"
)

// StatefulNotifier holds a value and notifies listeners when the value is updated.
// Unlike a Channel, it does not persist values, so a listener (calling Get)
// may not see all updates if multiple updates occur between calls to Get.
type StatefulNotifier[T any] struct {
	mu      sync.Mutex
	value   T
	updated chan struct{}
}

func NewStatefulNotifier[T any](initial T) *StatefulNotifier[T] {
	return &StatefulNotifier[T]{
		value:   initial,
		updated: make(chan struct{}),
	}
}

// Store updates the value and unblocks any listeners.
func (n *StatefulNotifier[T]) Store(value T) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.value = value
	old := n.updated
	n.updated = make(chan struct{})
	close(old)
}

// Load returns the current value, along with a channel that will unblock
// when the value is updated.
func (n *StatefulNotifier[T]) Load() (T, <-chan struct{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.value, n.updated
}

// Update will atomically provide the current value to the update function
// and store the result of the function.
// Note that this will call the user's function with a lock held, so
// if the function blocks, then other calls to the notifier will block.
func (n *StatefulNotifier[T]) Update(fn func(T) T) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.value = fn(n.value)
	old := n.updated
	n.updated = make(chan struct{})
	close(old)
}

// Wait blocks until the given condition function returns true
// or the context is canceled. It returns the value that satisfied the condition.
//
// Note that Wait may miss intermediate updates if multiple update occur quickly.
// If every update should be processed, use Channel instead.
func (n *StatefulNotifier[T]) Wait(ctx context.Context, fn func(T) bool) (T, error) {
	for {
		v, ch := n.Load()
		if fn(v) {
			return v, nil
		}

		// Wait for a change in state.
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-ch:
		}
	}
}
