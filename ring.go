package collections

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
	// right-hand side always contains the first element.
	if len(r.right) == 0 {
		var zero T
		return zero, false
	}

	el := r.right[0]
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
	if i < 0 || i >= r.Len() {
		var zero T
		return zero, false
	}

	idx := i - len(r.right)
	if idx >= 0 {
		// Shift elements to the left, which ensures that the end of the right
		// and the start of the left are adjacent (modulo ring size).
		el := r.left[idx]
		copy(r.left[idx:], r.left[idx+1:])
		r.left = r.left[:len(r.left)-1]
		return el, true
	}

	// Shift elements to the right, which ensures that the end of the right
	// and the start of the left are adjacent (modulo ring size).
	// Since i != 0 (handled above), there must be at least one element to shift.
	el := r.right[i]
	updated := r.right[1:]
	copy(updated, r.right[:i])
	r.right = updated
	return el, true
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
	n := min(len(out), r.Len())
	idx := copy(out, r.right)
	copy(out[idx:], r.left)
	return n
}

// Reset removes all elements from the ring.
func (r *Ring[T]) Reset() {
	r.left = r.elements[:0]
	r.right = r.elements[:0]
	clear(r.elements)
}
