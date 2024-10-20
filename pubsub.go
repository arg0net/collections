package collections

import (
	"context"
	"iter"
	"sync"
)

// Channel is a publish/subscribe channel. It is similar to an infinitely
// buffered Go channel, but where each value is sent to all subscribers.
//
// The zero value of a Channel is ready to use.
type Channel[T any] struct {
	mu   sync.Mutex // for reading `next` and for writes.
	next *message[T]
}

type message[T any] struct {
	value T
	next  *message[T]
	final chan struct{}
}

// Publish a new value to the channel. This value will be sent to all subscribers.
// Note that values are not persisted, so if no subscribers are listening when a
// value is published, it will be lost.
func (c *Channel[T]) Publish(value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.next == nil {
		// no subscribers, can drop message.
		return
	}

	next := &message[T]{final: make(chan struct{})}
	old := c.next
	c.next = next
	old.value = value
	old.next = next
	close(old.final)
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
func (c *Channel[T]) Watch(ctx context.Context, fn func(T) error) error {
	next := c.head()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-next.final:
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
func (c *Channel[T]) Receive() iter.Seq[T] {
	next := c.head()
	return func(yield func(T) bool) {
		for {
			select {
			case <-next.final:
				if !yield(next.value) {
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
	}

	go sub.loop(next, fn)
	return sub
}

type Subscription[T any] struct {
	once sync.Once
	stop chan struct{}
}

func (s *Subscription[T]) Cancel() {
	s.once.Do(func() { close(s.stop) })
}

func (s *Subscription[T]) loop(next *message[T], fn func(T)) {
	for {
		select {
		case <-s.stop:
			return

		case <-next.final:
			fn(next.value)
			next = next.next
		}
	}
}
