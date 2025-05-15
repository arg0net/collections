package collections

import (
	"context"
	"iter"
	"sync"
)

// Consume executes the given function for each element in the
// iterator in parallel, up to the specified number of workers.
//
// If a function returns an error, the context is cancelled and the error is
// returned.
//
// Once the input iterator is exhausted, the function waits for all workers to finish
// before returning. If none of the tasks returned an error, the function
// returns nil. If any task returned an error, the function returns the first
// error returned by a task.
func Consume[T any](ctx context.Context, maxWorkers int,
	in iter.Seq[T], fn func(context.Context, T) error,
) error {

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)
	subCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	done := subCtx.Done()

outer:
	for v := range in {
		select {
		case <-done:
			break outer
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(v T) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(subCtx, v); err != nil {
				cancel(err)
			}
		}(v)
	}

	wg.Wait()
	select {
	case <-done:
		return context.Cause(subCtx)
	default:
		cancel(nil)
		return nil
	}
}
