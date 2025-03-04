package objectpool

import (
	"math"
	"math/bits"
	"strconv"
	"sync"
	"unsafe"
)

func getSlice[T any](p uintptr) *SlicePool {
	return &getSliceSpl[T](p).SlicePool
}

func getSliceSpl[T any](p uintptr) *spl {
	index := (p >> offset) & maxIndex
	var ss []poolUintptr
	v := op.m[index].Load()
	if v != nil {
		ss = *v
		for i := range ss {
			if ss[i].uintptr == p {
				return &ss[i].slicePool
			}
		}
	}

	// lock for index conflict,
	lock := &op.ml[index]
	lock.Lock()
	v = op.m[index].Load()
	if v != nil {
		ss = *v
		for i := range ss {
			if ss[i].uintptr == p {
				lock.Unlock()
				return &ss[i].slicePool
			}
		}
	}

	l := len(ss)
	newSS := make([]poolUintptr, l+1)
	copy(newSS, ss)
	newSS[l] = poolUintptr{
		uintptr:   p,
		slicePool: spl{},
	}
	op.m[index].Store(&newSS)
	po := &newSS[l]
	lock.Unlock()
	return &po.slicePool
}

// Slice  is a slice object pool for T
// []T put sync.Pool is invalid
type Slice struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

type sliceType[T any] struct {
	_ [0]T
}

type SlicePool struct {
	pools [32]sync.Pool
}

func index(n uint32) uint32 {
	return uint32(bits.Len32(n - 1))
}

func getSlicePool[T any](s *SlicePool, cap int, minCap int) []T {
	if cap > math.MaxInt32 {
		return make([]T, 0, cap)
	}
	// byte Small memory allocation is too scattered. Starting from 128 bytes, the reuse rate is relatively high, other types start with len=16
	if cap < minCap {
		cap = minCap
	}

	idx := index(uint32(cap))
	if v := s.pools[idx].Get(); v != nil {
		bp := v.(unsafe.Pointer)
		x := (*int32)(bp)
		c := *x
		*x = 0
		s := &Slice{
			Data: bp,
			Len:  0,
			Cap:  int(c),
		}
		return *(*[]T)(unsafe.Pointer(s))
	}
	return make([]T, 0, 1<<idx)
}

func putSlicePool[T any](s *SlicePool, t []T) {
	t = t[:0]
	c := cap(t)
	if c < otherMinCap {
		return
	}
	idx := index(uint32(c))
	// []T not obtained by Get is placed in the Pool of the previous index
	if c != 1<<idx {
		idx--
	}
	slice := (*Slice)(unsafe.Pointer(&t))
	x := (*int32)(slice.Data)
	*x = int32(c)
	s.pools[idx].Put(slice.Data)
}

// GetSlice get a slice from object pool with T,len() == 0
func GetSlice[T any](cap int) []T {
	typPtr := GetPtr[sliceType[T]]()
	s := getSlice[T](typPtr)
	var minCap int
	if typPtr != bytesPtr {
		minCap = otherMinCap
	} else {
		minCap = byteMinCap
	}
	return getSlicePool[T](s, cap, minCap)
}

func GetSlice2[T any](s *SlicePool, cap int) []T {
	typPtr := GetPtr[sliceType[T]]()
	var minCap int
	if typPtr != bytesPtr {
		minCap = otherMinCap
	} else {
		minCap = byteMinCap
	}

	return getSlicePool[T](s, cap, minCap)
}

// GetSliceForSize  get a slice from object pool with T and size, len() == size
func GetSliceForSize[T any](size int) []T {
	s := GetSlice[T](size)
	s = s[:size]
	return s
}

func GetSliceForSize2[T any](sp *SlicePool, size int) []T {
	s := GetSlice2[T](sp, size)
	s = s[:size]
	return s
}

// PutSlice put a slice to object pool with T
func PutSlice[T any](t []T) {
	if cap(t) > math.MaxInt32 {
		return
	}
	typPtr := GetPtr[sliceType[T]]()
	if typPtr != bytesPtr {
		if cap(t) < otherMinCap {
			return
		}
	} else {
		if cap(t) < byteMinCap {
			return
		}
	}

	s := getSlice[T](typPtr)
	putSlicePool(s, t)
}

func PutSlice2[T any](s *SlicePool, t []T) {
	if cap(t) > math.MaxInt32 {
		return
	}
	typPtr := GetPtr[sliceType[T]]()
	if typPtr != bytesPtr {
		if cap(t) < otherMinCap {
			return
		}
	} else {
		if cap(t) < byteMinCap {
			return
		}
	}

	putSlicePool(s, t)
}

func PutSliceClear[T any](t []T) {
	if cap(t) > math.MaxInt32 {
		return
	}
	typPtr := GetPtr[sliceType[T]]()
	if typPtr != bytesPtr {
		if cap(t) < otherMinCap {
			return
		}
		clear(t)
	} else {
		if cap(t) < byteMinCap {
			return
		}
	}

	s := getSlice[T](typPtr)
	putSlicePool(s, t)
}

func PutSliceClear2[T any](s *SlicePool, t []T) {
	if cap(t) > math.MaxInt32 {
		return
	}
	typPtr := GetPtr[sliceType[T]]()
	if typPtr != bytesPtr {
		if cap(t) < otherMinCap {
			return
		}
		clear(t)
	} else {
		if cap(t) < byteMinCap {
			return
		}
	}

	putSlicePool(s, t)
}

// Bytes is a slice object pool for byte
type Bytes []byte

func GetBytes(cap int) Bytes {
	b := (Bytes)(getSlicePool[byte](bytesPool, cap, byteMinCap))
	b = b[:0]
	return b
}

func PutBytes(b Bytes) {
	PutSlice[byte](b)
}

func (b *Bytes) WriteString(s string) {
	*b = append(*b, s...)
}

func (b *Bytes) WriteBytes(c ...byte) {
	*b = append(*b, c...)
}

func (b *Bytes) String() string {
	return unsafe.String(unsafe.SliceData(*b), len(*b))
}

func (b *Bytes) WriteInt(i int64) {
	*b = strconv.AppendInt(*b, i, 10)
}

func (b *Bytes) WriteUint(i uint64) {
	*b = strconv.AppendUint(*b, i, 10)
}

func (b *Bytes) WriteFloat(f float64) {
	*b = strconv.AppendFloat(*b, f, 'g', -1, 64)
}

func (b *Bytes) WriteBool(v bool) {
	if v {
		b.WriteBytes('T')
	} else {
		b.WriteBytes('F')
	}
}

func (b *Bytes) Len() int {
	return len(*b)
}

func (b *Bytes) Cap() int {
	return cap(*b)
}

func (b *Bytes) Bytes() []byte {
	return *b
}

func (b *Bytes) Reset() {
	*b = (*b)[:0]
}
