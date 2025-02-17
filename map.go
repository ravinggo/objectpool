package objectpool

import (
	"sync"
	"unsafe"
)

type mapPtr[K comparable, V any] struct {
	_ [0]K
	_ [0]V
}

// GetMapPtr get type pointer of map[K]V
func GetMapPtr[K comparable, V any]() uintptr {
	var a any = (*mapPtr[K, V])(nil)
	t := *(**Type)(unsafe.Pointer(&a))
	return (uintptr)(unsafe.Pointer(t))
}

func getMap[K comparable, V any](p uintptr) *sync.Pool {
	return &getMapPl[K, V](p).Pool
}

func getMapPl[K comparable, V any](p uintptr) *pl {
	index := (p >> offset) & maxIndex
	var ss []poolUintptr
	v := op.m[index].Load()
	if v != nil {
		ss = *v
		for i := range ss {
			if ss[i].uintptr == p {
				return &ss[i].singlePool
			}
		}
	}

	lock := &op.ml[index]
	// lock for index conflict,
	lock.Lock()
	v = op.m[index].Load()
	if v != nil {
		ss = *v
		for i := range ss {
			if ss[i].uintptr == p {
				lock.Unlock()
				return &ss[i].singlePool
			}
		}
	}

	l := len(ss)
	newSS := make([]poolUintptr, l+1)
	copy(newSS, ss)
	newSS[l] = poolUintptr{
		uintptr: p,
		singlePool: pl{
			Pool: sync.Pool{
				New: func() any {
					return make(map[K]V, otherMinCap)
				},
			},
		},
	}
	op.m[index].Store(&newSS)
	po := &newSS[l]
	lock.Unlock()
	return &po.singlePool
}

// GetMap  get a map from object pool with K and V
func GetMap[K comparable, V any]() map[K]V {
	return getMap[K, V](GetMapPtr[K, V]()).Get().(map[K]V)
}

// PutMap put a map to object pool with K and V
func PutMap[K comparable, V any](t map[K]V) {
	clear(t)
	getMap[K, V](GetMapPtr[K, V]()).Put(t)
}
