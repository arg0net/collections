package collections

import (
	"context"
	"reflect"
)

type NotifierLoader[T any] interface {
	Load() (T, <-chan struct{})
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
func WaitAny[T any, N NotifierLoader[T]](ctx context.Context, fn func(T) bool,
	notifiers ...N) (T, int) {

	return WaitAnyMethod(ctx, fn, N.Load, notifiers...)
}

// WaitAnyMethod is like WaitAny, but takes a list of objects along with a
// method signature that returns a value and a notifier channel.
// This allows it to be used with similar operations which have a different
// method name or by using `method` as an adapter function.
func WaitAnyMethod[T any, V any](ctx context.Context,
	fn func(T) bool,
	method func(V) (T, <-chan struct{}),
	objs ...V) (T, int) {

	cases := make([]reflect.SelectCase, 0, len(objs)+1)
	for i, n := range objs {
		v, ch := method(n)
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
		if chosen == len(objs) {
			var zero T
			return zero, -1
		}

		v, ch := method(objs[chosen])
		if fn(v) {
			return v, chosen
		}
		cases[chosen].Chan = reflect.ValueOf(ch)
	}
}
