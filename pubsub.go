package collections

import (
	"context"
	"iter"
	"sync"
)

// Channel is a publish/subscribe channel. It is similar to a Go channel with
// infinite capacity, with a couple important differences.
//
// 1. Multiple receivers. There may be multiple receivers (or publishers), and
// all receivers get all messages.
//
// 2. Persistence. Messages are not persisted. If no receivers are listening when
// a message is published, it will be lost. When a receiver subscribes, it will
// only receive messages published after the subscription is created.
type Channel[T any] struct {
	mu   sync.Mutex // for reading `next` and for writes.
	next *message[T]
}

type message[T any] struct {
	value  T
	next   *message[T]
	final  chan struct{}
	closed bool
}

// Publish a new value to the channel. This value will be sent to all subscribers.
// Note that values are not persisted, so if no subscribers are listening when a
// value is published, it will be lost.
func (c *Channel[T]) Publish(value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.next == nil || c.next.closed {
		// drop message.
		return
	}

	next := &message[T]{final: make(chan struct{})}
	old := c.next
	c.next = next
	old.value = value
	old.next = next
	close(old.final)
}

// Close the channel. This will prevent any new values from being published, and
// will cause all subscribers to stop receiving values after the last message.
// For receive iterators, this will cause the iterator to terminate.
func (c *Channel[T]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.next == nil {
		c.next = &message[T]{final: make(chan struct{})}
	}
	if c.next.closed {
		return
	}
	c.next.closed = true
	close(c.next.final)
}

func (c *Channel[T]) head() *message[T] {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.next == nil {
		c.next = &message[T]{final: make(chan struct{})}
	}
	return c.next
}

// Watch updates on the channel. The function will be called with each new value
// sent to the channel. If the function returns an error, the subscription will
// be canceled and the error will be returned.
// If the channel is closed, Watch will return nil.
func (c *Channel[T]) Watch(ctx context.Context, fn func(T) error) error {
	next := c.head()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-next.final:
			if next.closed {
				return nil
			}
			if err := fn(next.value); err != nil {
				return err
			}
			next = next.next
		}
	}
}

// Receive subscribes to updates on the channel and returns a sequence of values.
// The subscription is setup before the function returns, so it is safe to publish
// values immediately after calling Receive.
// The sequence may be infinite, it will only terminate if the channel is closed.
func (c *Channel[T]) Receive() iter.Seq[T] {
	next := c.head()
	return func(yield func(T) bool) {
		for {
			select {
			case <-next.final:
				if next.closed || !yield(next.value) {
					return
				}
				next = next.next
			}
		}
	}
}

// Subscribe is like Watch, but without the context. The subscription will run
// until it is canceled.
// The subscription is setup before the function returns, so it is safe to
// publish values immediately after calling Subscribe.
func (c *Channel[T]) Subscribe(fn func(T)) *Subscription[T] {
	next := c.head()
	sub := &Subscription[T]{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	go sub.loop(next, fn)
	return sub
}

// Subscription is a subscription to a Channel. It will receive all values
// published to the channel until it is canceled.
type Subscription[T any] struct {
	once sync.Once     // to ensure stop is closed only once.
	stop chan struct{} // close to stop the subscription loop.
	done chan struct{} // closed when the subscription loop has finished.
}

// Cancel the subscription. This will cause the subscription to stop receiving
// updates from the channel.
// Note that the subscription loop runs in the background, so there may
// be some latency between the cancel call and the subscription stopping.
func (s *Subscription[T]) Cancel() {
	s.once.Do(func() { close(s.stop) })
}

// Done returns a channel that will be closed when the subscription loop has
// finished.
func (s *Subscription[T]) Done() <-chan struct{} {
	return s.done
}

func (s *Subscription[T]) loop(next *message[T], fn func(T)) {
	defer close(s.done)
	for {
		select {
		case <-s.stop:
			return

		case <-next.final:
			if next.closed {
				return
			}
			fn(next.value)
			next = next.next
		}
	}
}
