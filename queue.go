package collections

import (
	"context"
	"iter"
	"sync"
)

// Queue is a generic interface that represents a queue of items.
type Queue[T any] interface {
	// Enqueue adds an item to the queue.
	Enqueue(item T)

	// Dequeue removes an item from the queue.
	// Returns the item and a boolean indicating if the item was successfully removed.
	Dequeue() (T, bool)

	// Peek returns the item at the front of the queue without removing it.
	// Returns the item and a boolean indicating if the item was successfully retrieved.
	Peek() (T, bool)

	// IsEmpty returns true if the queue is empty.
	IsEmpty() bool

	// Size returns the number of items in the queue.
	Size() int

	// Clear removes all items from the queue.
	Clear()

	// Wait blocks until an item is available.
	Wait(ctx context.Context) error

	// All returns an iterator over the queue.
	// The iterator will return the items in the order they were added to the queue.
	// Iteration blocks when the queue is empty.
	All(ctx context.Context) iter.Seq[T]
}

// NewQueue creates a new queue.
func NewQueue[T any]() Queue[T] {
	return &queue[T]{
		available: make(chan struct{}),
	}
}

type queue[T any] struct {
	items     []T
	mu        sync.Mutex
	available chan struct{} // used to signal that elements are available
}

func (q *queue[T]) Enqueue(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		close(q.available)
	}
	q.items = append(q.items, item)
}

func (q *queue[T]) Dequeue() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	item := q.items[0]
	if len(q.items) == 1 {
		q.available = make(chan struct{})
		q.items = nil
	} else {
		q.items = q.items[1:]
	}
	return item, true
}

func (q *queue[T]) Peek() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	return q.items[0], true
}

func (q *queue[T]) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items) == 0
}

func (q *queue[T]) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *queue[T]) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) > 0 {
		q.items = q.items[:0]
		q.available = make(chan struct{})
	}
}

func (q *queue[T]) Wait(ctx context.Context) error {
	q.mu.Lock()
	available := q.available
	q.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-available:
		return nil
	}
}

func (q *queue[T]) All(ctx context.Context) iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			if err := q.Wait(ctx); err != nil {
				return
			}
			item, ok := q.Dequeue()
			if !ok {
				continue
			}
			if !yield(item) {
				return
			}
		}
	}
}
