package collections_test

import (
	"slices"
	"testing"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
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

func TestRingScan(t *testing.T) {
	r := collections.NewRing[int](7)
	for i := 0; i < 4; i++ {
		r.PushBack(i)
	}
	for i := 5; i < 100; i++ {
		check := i % 4
		value, ok := r.PeekIndex(check)
		require.True(t, ok)
		found, idx := r.Scan(func(v int) bool {
			return v == value
		})
		require.Equal(t, value, found)
		require.Equal(t, check, idx)
		r.PopFront()
		r.PushBack(i)
	}

	// Final result should be 96, 97, 98, 99
	require.Equal(t, []int{96, 97, 98, 99}, slices.Collect(r.All()))
}

func TestRingResize(t *testing.T) {
	r := collections.NewRing[int](3)
	require.True(t, r.PushBack(1))
	require.True(t, r.PushBack(2))
	require.True(t, r.PushBack(3))
	require.False(t, r.PushBack(4))
	require.Error(t, r.Resize(2))
	require.NoError(t, r.Resize(5))
	require.Equal(t, 3, r.Len())
	require.Equal(t, 5, r.Cap())
}

func TestRingCopy(t *testing.T) {
	r := collections.NewRing[int](3)
	require.Equal(t, 2, r.PushBackAll([]int{1, 2}))
	require.Equal(t, 1, r.PushBackAll([]int{3, 4}))
	require.Equal(t, 0, r.PushBackAll([]int{5, 6}))

	buf := make([]int, 3)
	require.Equal(t, 3, r.Copy(buf))
	require.Equal(t, []int{1, 2, 3}, buf)

	r.PopFront()
	require.Equal(t, 2, r.Copy(buf))
	require.Equal(t, []int{2, 3}, buf[:2])

	require.Equal(t, 1, r.PushBackAll([]int{5, 6}))
	require.Equal(t, 3, r.Copy(buf))
	require.Equal(t, []int{2, 3, 5}, buf)
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

// fakeRing is a simplified implementation of a buffer used for fuzzing tests.
// This behaves like a ring buffer, but it's not optimized for performance.
type fakeRing struct {
	elements []int
}

func (r *fakeRing) PushBack(e int) bool {
	if len(r.elements) == cap(r.elements) {
		return false
	}
	r.elements = append(r.elements, e)
	return true
}

func (r *fakeRing) PushBackAll(in []int) int {
	n := min(len(in), cap(r.elements)-len(r.elements))
	r.elements = append(r.elements, in[:n]...)
	return n
}

func (r *fakeRing) PopFront() (int, bool) {
	if len(r.elements) == 0 {
		return 0, false
	}
	el := r.elements[0]
	copy(r.elements, r.elements[1:])
	r.elements = r.elements[:len(r.elements)-1]
	return el, true
}

func (r *fakeRing) Copy(out []int) int {
	return copy(out, r.elements)
}

func (r *fakeRing) Len() int {
	return len(r.elements)
}

func (r *fakeRing) PopIndex(i int) (int, bool) {
	if i < 0 || i >= len(r.elements) {
		return 0, false
	}
	el := r.elements[i]
	r.elements = append(r.elements[:i], r.elements[i+1:]...)
	return el, true
}

func (r *fakeRing) PeekIndex(idx int) (int, bool) {
	if idx < 0 || idx >= len(r.elements) {
		return 0, false
	}
	return r.elements[idx], true
}

type ringOp int

const (
	pushBack ringOp = iota
	popFront
	popIndex
	peekIndex
	scan
	copyOut
	pushAll
	lastOpForCounting // keep last
)

func dup[T any](s []T) []T {
	out := make([]T, len(s))
	copy(out, s)
	return out
}

func FuzzRing(f *testing.F) {
	init := []int{1, 2, 3, 4, 5}

	f.Fuzz(func(t *testing.T, data []byte) {
		fake := &fakeRing{elements: dup(init)}
		real := collections.NewRing[int](len(init))
		for _, v := range init {
			real.PushBack(v)
		}

		fz := fuzz.NewConsumer(data)
		var ops []ringOp
		err := fz.CreateSlice(&ops)
		if err != nil {
			return
		}

		var buf1, buf2 [5]int
		for i := 0; i < len(ops); i++ {
			switch ops[i] % lastOpForCounting {
			case pushBack:
				var value int
				if i+1 < len(ops) {
					value = int(ops[i+1])
					i++
				}
				t.Logf("pushBack %d", value)
				ok1 := fake.PushBack(value)
				ok2 := real.PushBack(value)
				if ok1 != ok2 {
					t.Fatalf("pushBack differs: %v vs %v in %v vs %v", ok1, ok2, fake, real)
				}

			case popFront:
				t.Logf("popFront")
				f1, ok1 := fake.PopFront()
				r1, ok2 := real.PopFront()
				if f1 != r1 || ok1 != ok2 {
					t.Fatalf("popFront differs: %v vs %v in %v vs %v", f1, r1, fake, real)
				}

			case popIndex:
				var idx int
				if i+1 < len(ops) {
					idx = int(ops[i+1])
					i++
				}
				t.Logf("popIndex %d", idx)
				f1, ok1 := fake.PopIndex(idx)
				r1, ok2 := real.PopIndex(idx)
				if f1 != r1 || ok1 != ok2 {
					t.Fatalf("popIndex differs: %v vs %v in %v vs %v", f1, r1, fake, real)
				}

			case peekIndex:
				var idx int
				if i+1 < len(ops) {
					idx = int(ops[i+1])
					i++
				}
				t.Logf("peekIndex %d", idx)
				f1, ok1 := fake.PeekIndex(idx)
				r1, ok2 := real.PeekIndex(idx)
				if f1 != r1 || ok1 != ok2 {
					t.Fatalf("peekIndex differs: %v vs %v in %v vs %v", f1, r1, fake, real)
				}

			case scan:
				var idx int
				if i+1 < len(ops) {
					idx = int(ops[i+1])
					i++
				}
				t.Logf("scan %d", idx)
				scanNum := 0
				v, loc := real.Scan(func(v int) bool {
					o := scanNum
					scanNum++
					return o == idx
				})
				v2, ok2 := real.PeekIndex(idx)
				if ok2 && (loc != idx || v != v2) {
					t.Fatalf("scan differs: %v vs %v in %v", v, v2, real)
				}

			case copyOut:
				var n int
				if i+1 < len(ops) {
					n = min(int(ops[i+1]), len(buf1))
					i++
				}
				t.Logf("copyOut %d", n)
				fake.Copy(buf1[:n])
				real.Copy(buf2[:n])

			case pushAll:
				var n int
				if i+1 < len(ops) {
					n = min(int(ops[i+1]), len(buf1))
					i++
				}
				t.Logf("pushAll %d", n)
				fake.PushBackAll(buf1[:n])
				real.PushBackAll(buf2[:n])

			}
			if fake.Copy(buf1[:]) != real.Copy(buf2[:]) {
				t.Fatalf("copy differs")
			}
			if buf1 != buf2 {
				t.Fatalf("buffers differ: %v vs %v", buf1, buf2)
			}
		}
	})
}
