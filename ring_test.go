package collections_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arg0net/collections"
)

func TestRing(t *testing.T) {
	r := collections.NewRing[int](3)

	buf := make([]int, 5)

	require.Equal(t, 0, r.Len())
	require.Equal(t, 0, r.Copy(buf))
	require.True(t, r.PushBack(1))
	require.Equal(t, 1, r.Copy(buf))
	require.Equal(t, []int{1}, buf[:1])
	require.True(t, r.PushBack(2))
	require.Equal(t, 2, r.Copy(buf))
	require.Equal(t, []int{1, 2}, buf[:2])
	require.True(t, r.PushBack(3))
	require.False(t, r.PushBack(4))

	el, ok := r.PopFront()
	require.True(t, ok)
	require.Equal(t, 1, el)
	require.Equal(t, 2, r.Copy(buf))
	require.Equal(t, []int{2, 3}, buf[:2])

	require.True(t, r.PushBack(4))
	require.False(t, r.PushBack(5))
	require.Equal(t, 3, r.Copy(buf))
	require.Equal(t, []int{2, 3, 4}, buf[:3])
	require.Equal(t, 3, r.Len())
	require.Equal(t, 3, r.Cap())

	el, ok = r.PopFront()
	require.True(t, ok)
	require.Equal(t, 2, el)

	el, ok = r.PopFront()
	require.True(t, ok)
	require.Equal(t, 3, el)

	el, ok = r.PopFront()
	require.True(t, ok)
	require.Equal(t, 4, el)

	el, ok = r.PopFront()
	require.False(t, ok)
	require.Equal(t, 0, el)
}

func TestRingIndex(t *testing.T) {
	r := collections.NewRing[int](5)
	buf := make([]int, 5)

	require.True(t, r.PushBack(1))
	require.True(t, r.PushBack(2))
	require.True(t, r.PushBack(3))
	require.True(t, r.PushBack(4))
	require.True(t, r.PushBack(5))

	el, ok := r.PopIndex(0) // 1,2,3,4,5
	require.True(t, ok)
	require.Equal(t, 1, el)
	require.Equal(t, 4, r.Copy(buf))
	require.Equal(t, []int{2, 3, 4, 5, 0}, buf)

	require.True(t, r.PushBack(6))
	el, ok = r.PopIndex(1) // 2,3,4,5,6
	require.True(t, ok)
	require.Equal(t, 3, el)
	require.Equal(t, 4, r.Copy(buf))
	require.Equal(t, []int{2, 4, 5, 6, 0}, buf)

	require.True(t, r.PushBack(7))
	el, ok = r.PopIndex(2) // 2,4,5,6,7
	require.True(t, ok)
	require.Equal(t, 5, el)
	require.Equal(t, 4, r.Copy(buf))
	require.Equal(t, []int{2, 4, 6, 7, 0}, buf)

	require.True(t, r.PushBack(8))
	el, ok = r.PopIndex(3) // 2,4,6,7,8
	require.True(t, ok)
	require.Equal(t, 7, el)
	require.Equal(t, 4, r.Copy(buf))
	require.Equal(t, []int{2, 4, 6, 8, 0}, buf)

	require.True(t, r.PushBack(9))
	el, ok = r.PopIndex(4) // 2,4,6,8,9
	require.True(t, ok)
	require.Equal(t, 9, el)

	require.Equal(t, 4, r.Copy(buf))
	require.Equal(t, []int{2, 4, 6, 8, 0}, buf)
}

func TestRingIndex_Wrap(t *testing.T) {
	r := collections.NewRing[int](3)
	r.PushBack(1)
	r.PushBack(2)
	r.PushBack(3)
	el, ok := r.PopIndex(2)
	require.True(t, ok)
	require.Equal(t, 3, el)
	r.PushBack(4)
	el, ok = r.PopIndex(1)
	require.True(t, ok)
	require.Equal(t, 2, el)
	r.PushBack(5)
	el, ok = r.PopIndex(0)
	require.True(t, ok)
	require.Equal(t, 1, el)
	r.PushBack(6)
	el, ok = r.PopIndex(2)
	require.True(t, ok)
	require.Equal(t, 6, el)
}

func BenchmarkRing(b *testing.B) {
	r := collections.NewRing[int](1024)
	// fill the ring
	var nextWrite int
	var nextRead int
	for i := 0; i < 1024; i++ {
		r.PushBack(nextWrite)
		nextWrite++
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, ok := r.PopFront()
		if !ok || v != nextRead {
			b.Fatalf("expected %d, got %d", nextRead, v)
		}
		nextRead++

		r.PushBack(nextWrite)
		nextWrite++
	}
}
