# collections

Golang Data Collections

## Ring Buffer

The `Ring[T]` type provide a fixed-size ring buffer, implemented as a slice of
type `T` . This was created because of edge-case bugs found in a free Ring
implementation.

It uses views into a slice rather than doing offset arithmetic, which simplifies
the implementation and improves performance. An incremental fuzzer test is used
to search for unexpected edge cases by comparing against an unoptimized naive
implementation.

To create a new ring buffer, use NewRing:

```go
r := NewRing[*MyData](numElements)
if !r.PushBack(&MyData{...}) {
    // the ring buffer is full!
}

v, ok := r.PopFront()
if !ok {
    // no elements left in ring buffer
}

```

`Peek` operations are also provided, which look at the next element, or even an
arbitrary index, without modifying the ring. The `Len` and `Cap` functions
indicate the size and capacity of the ring.

### StatefulNotifier

`StatefulNotifier[T]` acts as an atomic variable which allows waiting on a state
change. The `Load` method atomically reads the current value and returns an
update notifier which is signalled when the state changes. This allows for
efficient wait loops without by avoiding polling for updates.

There are multiple types of wait mechanisms, including `Wait` which blocks until
an arbitrary condition is met, and `Watch` which turns the updates into a
range-friendly sequence. Note that this is not an update queue, meaning that
individual updates are not stored (only the last value is available). A receiver
is guaranteed to receive the latest value, but is not guaranteed to see all
intermediate updates if it hasn't kept up with the state changes.

### PubSub

`Channel[T]` provides a publish/subscribe channel. This is similar to a native
`chan T`, except that it has infinite capacity and allows for multiple
subscribers.

To create, simplify define a `var ch Channel[*myValueType]`. The `Publish`
method adds a new value to the channel. To subscribe, use `Subscribe` to create
a subscription object, which will call the callback in the background.
Alternatively use `Watch` to apply updates in the foreground, or `Receive` to
return an `iter.Seq` which can be iterated over using `range`.

Note that updates are not persisted. As soon as all active subscribers have read
a message, then it is no longer accessible and will be garbage collected. This
means that if a channel is created and values are published before any
subscribers have been created, then those values will disappear immediately.
