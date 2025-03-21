package collections

import (
	"fmt"
	"io"
	"iter"
)

// Ring is a fixed-size ring buffer that supports pushing and popping elements,
// as well as copying elements into a slice, and removing an element by index.
// The ring is implemented as a single slice, which is never reallocated.
//
// Note that no synchronization is done. If the ring is accessed concurrently,
// it must be synchronized externally.
type Ring[T any] struct {
	elements []T

	// Left and right are slices of the elements slice.
	left  []T // left half of the ring, when right is full and the ring wraps.
	right []T // right half of the ring, containing start.
}

// NewRing creates a new ring buffer with the given fixed size.
func NewRing[T any](fixedSize int) *Ring[T] {
	elements := make([]T, fixedSize)
	return &Ring[T]{
		elements: elements,
		left:     elements[:0],
		right:    elements[:0],
	}
}

// PushBack adds the element to the ring. If the ring is full, it returns false.
func (r *Ring[T]) PushBack(e T) bool {
	switch {
	case cap(r.right) > len(r.right):
		r.right = append(r.right, e)
	case len(r.left)+len(r.right) == cap(r.elements):
		return false // ring is full
	default:
		// right side is full, so wrapping around on the left side.
		r.left = append(r.left, e)
	}
	return true
}

// PopFront removes and returns the first element in the ring.
// If the ring is empty, it returns false.
func (r *Ring[T]) PopFront() (T, bool) {
	var zero T
	// right-hand side always contains the first element.
	if len(r.right) == 0 {
		return zero, false
	}

	el := r.right[0]
	r.right[0] = zero
	r.right = r.right[1:]
	if cap(r.right) == 0 {
		// right side is exhausted, so what was the left is now the right.
		r.right = r.left
		r.left = r.elements[:0]
	}
	return el, true
}

// PopIndex removes and returns the element at the given index.
// This will require copying elements to maintain the ring structure, which
// has a time complexity of O(n) in the worst case.
//
// If the index is out of bounds, it returns false.
// The index is 0-based, with 0 being the first element in the ring.
// PopIndex(0) is equivalent to PopFront.
func (r *Ring[T]) PopIndex(i int) (T, bool) {
	if i == 0 {
		return r.PopFront()
	}
	var zero T
	if i < 0 || i >= r.Len() {
		return zero, false
	}

	idx := i - len(r.right)
	if idx >= 0 {
		// Shift elements to the left, which ensures that the end of the right
		// and the start of the left are adjacent (modulo ring size).
		el := r.left[idx]
		copy(r.left[idx:], r.left[idx+1:])
		r.left[len(r.left)-1] = zero
		r.left = r.left[:len(r.left)-1]
		return el, true
	}

	// Shift elements to the right, which ensures that the end of the right
	// and the start of the left are adjacent (modulo ring size).
	// Since i != 0 (handled above), there must be at least one element to shift.
	el := r.right[i]
	updated := r.right[1:]
	copy(updated, r.right[:i])
	r.right[0] = zero
	r.right = updated
	return el, true
}

// PeekFront returns the first element in the ring without removing it.
func (r *Ring[T]) PeekFront() (T, bool) {
	if len(r.right) == 0 {
		var zero T
		return zero, false
	}
	return r.right[0], true
}

// PeekIndex returns the element at the given index without removing it.
// If the index is out of bounds, it returns false.
// The index is 0-based, with 0 being the first element in the ring.
// PeekIndex(0) is equivalent to PeekFront.
func (r *Ring[T]) PeekIndex(i int) (T, bool) {
	if i == 0 {
		return r.PeekFront()
	}
	if i < 0 || i >= r.Len() {
		var zero T
		return zero, false
	}

	idx := i - len(r.right)
	if idx >= 0 {
		return r.left[idx], true
	}
	return r.right[i], true
}

// Len returns the number of elements in the ring.
func (r *Ring[T]) Len() int {
	return len(r.left) + len(r.right)
}

// Cap returns the fixed size of the ring. This is constant for the lifetime of the ring.
func (r *Ring[T]) Cap() int {
	return cap(r.elements)
}

// Copy makes a copy of the first n elements of the ring into the out slice.
// It returns the number of elements copied.
// This does not consume elements from the ring.
func (r *Ring[T]) Copy(out []T) int {
	idx := copy(out, r.right)
	return idx + copy(out[idx:], r.left)
}

// Read copies the first n elements from the ring into the out slice.
// It returns the number of elements copied and an error if the ring is empty.
// If the ring is a Ring[byte], then this implements io.Reader.
func (r *Ring[T]) Read(out []T) (int, error) {
	n := r.Copy(out)
	if n == 0 {
		return 0, io.EOF
	}

	// consume the first n elements
	r.Drop(n)
	return n, nil
}

// Write writes the elements to the ring from the in slice.
// It returns the number of elements written and an error if the ring is full.
// If the ring is a Ring[byte], then this implements io.Writer.
func (r *Ring[T]) Write(in []T) (int, error) {
	available := r.Cap() - r.Len()
	if available == 0 {
		return 0, io.ErrShortWrite
	}

	written := 0
	expected := len(in)
	toWrite := min(available, expected)

	// First fill the right side if it has space
	if rightSpace := cap(r.right) - len(r.right); rightSpace > 0 {
		n := min(rightSpace, toWrite)
		r.right = append(r.right, in[:n]...)
		written += n
		toWrite -= n
		in = in[n:]
	}

	// Then fill the left side if we still have elements to write
	if toWrite > 0 && len(r.left)+len(r.right) < cap(r.elements) {
		r.left = append(r.left, in[:toWrite]...)
		written += toWrite
	}

	if written < expected {
		return written, io.ErrShortWrite
	}
	return written, nil
}

// Drop removes the first n elements from the ring.
// If n is greater than the number of elements in the ring, all elements are removed.
func (r *Ring[T]) Drop(n int) {
	if n >= r.Len() {
		// If dropping more elements than we have, just reset
		r.Reset()
		return
	}

	// First drop from right side
	if n < len(r.right) {
		r.right = r.right[n:]
		return
	}

	// Dropped all of right, now drop from left
	n -= len(r.right)
	r.right = r.elements[:len(r.left)-n]
	r.left = r.elements[:0]
}

// Resize changes the size of the ring.
// The new size must be greater than or equal to the current size.
func (r *Ring[T]) Resize(newSize int) error {
	if newSize < r.Len() {
		return fmt.Errorf("new size %d is too small to hold %d elements", newSize, r.Len())
	}

	els := make([]T, newSize)
	count := r.Copy(els)
	r.left = els[:count]
	r.right = els[:0]
	r.elements = els
	return nil
}

// Reset removes all elements from the ring.
func (r *Ring[T]) Reset() {
	r.left = r.elements[:0]
	r.right = r.elements[:0]
	clear(r.elements)
}

// Scan calls the given function for each element in the ring, in order.
// If the function returns true, then the value and index of the element are returned.
// If no match is found, then returns the zero value of T and -1.
func (r *Ring[T]) Scan(fn func(T) bool) (T, int) {
	for i, e := range r.right {
		if fn(e) {
			return e, i
		}
	}
	for i, e := range r.left {
		if fn(e) {
			return e, i + len(r.right)
		}
	}
	var zero T
	return zero, -1
}

// All returns a sequence of all elements in the ring.
func (r *Ring[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, e := range r.right {
			if !yield(e) {
				return
			}
		}
		for _, e := range r.left {
			if !yield(e) {
				return
			}
		}
	}
}
