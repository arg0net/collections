package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSkipZerosElements tests that Skip correctly zeros elements in the ring.
// Internal details are tested here, so must be in the collections package.
func TestSkipZerosElements(t *testing.T) {
	r := NewRing[*int](5)

	// Push 4 elements
	for i := 1; i <= 4; i++ {
		v := i
		require.True(t, r.PushBack(&v), "push should succeed")
	}

	skipped := r.Skip(3)
	require.Equal(t, 3, skipped, "should skip 3 elements")

	// Verify zeroed elements in underlying storage
	for i := 0; i < 3; i++ {
		require.Nil(t, r.elements[i], "element %d should be nil", i)
	}

	// Verify remaining element
	require.NotNil(t, r.elements[3], "remaining element should exist")
	require.Equal(t, 4, *r.elements[3], "remaining element value should be 4")

	// Verify unused capacity is nil
	require.Nil(t, r.elements[4], "unused element should be nil")
}
