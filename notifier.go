package collections

import (
	"context"
	"reflect"
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

// WaitAny blocks until one of the given states match the condition function,
// or else the context is canceled. It returns the value that satisfied the condition,
// along with an index of the notifier that was matched.
//
// Note that, like Wait, WaitAny may miss intermediate updates if multiple
// updates occur quickly.
//
// If the context was canceled, the value will be the zero value and the
// index will be -1.
func WaitAny[T any](ctx context.Context, fn func(T) bool,
	notifiers ...*StatefulNotifier[T]) (T, int) {

	cases := make([]reflect.SelectCase, 0, len(notifiers)+1)
	for i, n := range notifiers {
		v, ch := n.Load()
		if fn(v) {
			return v, i
		}
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	})

	for {
		chosen, _, _ := reflect.Select(cases)
		if chosen == len(notifiers) {
			var zero T
			return zero, -1
		}

		v, ch := notifiers[chosen].Load()
		if fn(v) {
			return v, chosen
		}
		cases[chosen].Chan = reflect.ValueOf(ch)
	}
}
