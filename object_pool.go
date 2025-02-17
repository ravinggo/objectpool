package objectpool

import (
	"math"
	"sync"
	"sync/atomic"
)

const (
	maxIndex    = math.MaxUint16 - 1
	otherMinCap = 16
	byteMinCap  = 128
	KindMask    = (1 << 5) - 1
	offset      = 5
)

var (
	op       = objectPool{}
	bytesPtr = func() uintptr {
		return GetPtr[sliceType[byte]]()
	}()
	bytesPool = getSlice[byte](bytesPtr)
)

// GetTypePool get a sync.Pool with T
func GetTypePool[T any]() *sync.Pool {
	return get[T](GetPtr[T]())
}

// GetSliceTypePool get a SlicePool with T
func GetSliceTypePool[T any]() *SlicePool {
	return getSlice[T](GetPtr[sliceType[T]]())
}

// GetMapTypePool get a sync.Pool with map[K]V
func GetMapTypePool[K comparable, V any]() *sync.Pool {
	return getMap[K, V](GetMapPtr[K, V]())
}

type spl struct {
	SlicePool
}
type pl struct {
	sync.Pool
}

type poolUintptr struct {
	uintptr
	slicePool  spl
	singlePool pl
}

type objectPool struct {
	m  [math.MaxUint16]atomic.Pointer[[]poolUintptr]
	ml [math.MaxUint16]sync.Mutex
}
