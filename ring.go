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

// Limit returns a view of the Ring which is limited to the given number of elements.
// If the ring has fewer elements than the limit, then it does nothing.
// Note that this shares the underlying data with the original ring, so
// modifying one will cause unpredictable results in the other.
func (r *Ring[T]) Limit(limit int) *Ring[T] {
	if r.Len() <= limit {
		return r
	}

	extra := limit - len(r.right)
	if extra <= 0 {
		return &Ring[T]{
			elements: r.elements,
			right:    r.right[:limit],
			left:     r.elements[:0],
		}
	}

	return &Ring[T]{
		elements: r.elements,
		right:    r.right,
		left:     r.left[:extra],
	}
}

// PushBatch adds the elements in the array to the ring, in order.
// It returns the number of elements added.
func (r *Ring[T]) PushBatch(arr []T) int {
	var added int
	if cap(r.right) > len(r.right) {
		n := copy(r.right[len(r.right):cap(r.right)], arr)
		r.right = r.right[:len(r.right)+n]
		arr = arr[n:]
		added += n
	}
	if len(arr) == 0 {
		return added
	}

	mmax := min(len(arr), len(r.elements)-len(r.right)-len(r.left))
	added += copy(r.elements[len(r.left):], arr[:mmax])

	return added
}

// WriteTo copies the elements from the ring to the given ring.
func (r *Ring[T]) WriteTo(out *Ring[T]) int {
	copied := out.PushBatch(r.right)
	if len(r.left) > 0 && copied == len(r.right) {
		copied += out.PushBatch(r.left)
	}
	return copied
}

// Compact causes the ring to compact the elements to the front of the space
// as one contiguous slice.
func (r *Ring[T]) Compact() {
	if len(r.left) == 0 {
		// Simple case - move the right-hand-side elements.
		copy(r.elements, r.right)
		clear(r.elements[len(r.right):])
		r.right = r.elements[:len(r.right)]
		return
	}

	// Use temporary space to avoid the headache of shifting in place.
	size := r.Len()
	oldLeft := make([]T, len(r.left))
	copy(oldLeft, r.left)                    // make a copy of left side
	copy(r.elements, r.right)                // copy right side into position
	copy(r.elements[len(r.right):], oldLeft) // put left side after right side
	clear(r.elements[size:])                 // clear the rest of the elements
	r.right = r.elements[:size]
	r.left = r.elements[:0]
}
