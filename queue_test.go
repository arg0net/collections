package collections

import (
	"context"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		q := NewQueue[int]()

		if !q.IsEmpty() {
			t.Error("new queue should be empty")
		}

		if _, ok := q.Peek(); ok {
			t.Error("peek on empty queue should return false")
		}

		if _, ok := q.Dequeue(); ok {
			t.Error("dequeue on empty queue should return false")
		}

		q.Enqueue(1)
		q.Enqueue(2)
		q.Enqueue(3)

		if q.IsEmpty() {
			t.Error("queue should not be empty after enqueuing")
		}

		if size := q.Size(); size != 3 {
			t.Errorf("expected size 3, got %d", size)
		}

		if val, ok := q.Peek(); !ok || val != 1 {
			t.Errorf("peek should return 1, got %v, %v", val, ok)
		}

		val, ok := q.Dequeue()
		if !ok || val != 1 {
			t.Errorf("dequeue should return 1, got %v, %v", val, ok)
		}

		val, ok = q.Dequeue()
		if !ok || val != 2 {
			t.Errorf("dequeue should return 2, got %v, %v", val, ok)
		}

		if size := q.Size(); size != 1 {
			t.Errorf("expected size 1, got %d", size)
		}

		q.Clear()
		if !q.IsEmpty() {
			t.Error("queue should be empty after clear")
		}
	})

	t.Run("wait operation", func(t *testing.T) {
		q := NewQueue[string]()
		ctx := context.Background()

		// Test wait with immediate value
		q.Enqueue("test")
		if err := q.Wait(ctx); err != nil {
			t.Errorf("wait should not error when queue has items: %v", err)
		}

		q.Clear()

		// Test wait with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		done := make(chan struct{})
		go func() {
			err := q.Wait(ctx)
			if err == nil {
				t.Error("wait should timeout when queue is empty")
			}
			close(done)
		}()

		select {
		case <-done:
			// Test passed
		case <-time.After(200 * time.Millisecond):
			t.Error("wait did not timeout as expected")
		}
	})

	t.Run("iterator", func(t *testing.T) {
		q := NewQueue[int]()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Add some items
		expected := []int{1, 2, 3, 4, 5}
		for _, v := range expected {
			q.Enqueue(v)
		}

		// Collect items from iterator
		var result []int
		for v := range q.All(ctx) {
			result = append(result, v)
			if len(result) == len(expected) {
				cancel()
			}
		}

		// Verify results
		if len(result) != len(expected) {
			t.Errorf("expected %d items, got %d", len(expected), len(result))
		}
		for i, v := range result {
			if v != expected[i] {
				t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
			}
		}
	})
}
