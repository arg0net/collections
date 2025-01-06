package collections

import (
	"context"
	"iter"
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

// NewStatefulNotifier creates a new StatefulNotifier with the given initial value.
func NewStatefulNotifier[T any](initial T) *StatefulNotifier[T] {
	return &StatefulNotifier[T]{
		value: initial,
	}
}

// Store updates the value and unblocks any listeners.
func (n *StatefulNotifier[T]) Store(value T) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.value = value
	if n.updated != nil {
		close(n.updated)
		n.updated = nil
	}
}

// Load returns the current value, along with a channel that will unblock
// when the value is updated.
func (n *StatefulNotifier[T]) Load() (T, <-chan struct{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.updated == nil {
		n.updated = make(chan struct{})
	}
	return n.value, n.updated
}

// Update will atomically provide the current value to the update function
// and store the result of the function.
// Note that this will call the user's function with a lock held, so
// if the function blocks, then other calls to the notifier will block.
func (n *StatefulNotifier[T]) Update(fn func(T) T) T {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.value = fn(n.value)
	if n.updated != nil {
		close(n.updated)
		n.updated = nil
	}
	return n.value
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

// Watch returns an iterator which will yield the current value and any updates.
// Note that updates may be missed if multiple updates occur quickly.
// If all updates should be processed, use a Channel instead.
// If the context is cancelled, then the iterator terminates.
func (n *StatefulNotifier[T]) Watch(ctx context.Context) iter.Seq[T] {
	v, ch := n.Load()
	return func(yield func(T) bool) {
		for {
			if !yield(v) {
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-ch:
				v, ch = n.Load()
			}
		}
	}
}
