package objectpool

import (
	"reflect"
	"sync"
	"unsafe"
)

// GetPtr get type pointer of T
func GetPtr[T any]() uintptr {
	var a any = (*T)(nil)
	t := *(**Type)(unsafe.Pointer(&a))
	if t.Kind_&KindMask == uint8(reflect.Pointer) {
		t = (*PtrType)(unsafe.Pointer(t)).Elem
	}
	return (uintptr)(unsafe.Pointer(t))
}

func GetPtrAndIndex[T any]() (uintptr, int) {
	p := GetPtr[T]()
	index := int(p>>offset) & maxIndex
	return p, index
}

// GetPtrAny get type pointer of param a
func GetPtrAny(a any) uintptr {
	t := *(**Type)(unsafe.Pointer(&a))
	if t.Kind_&KindMask == uint8(reflect.Pointer) {
		t = (*PtrType)(unsafe.Pointer(t)).Elem
	}
	return (uintptr)(unsafe.Pointer(t))
}

func GetPtrAnyAndIndex(a any) (uintptr, int) {
	p := GetPtrAny(a)
	index := int(p>>offset) & maxIndex
	return p, index
}

func get[T any](p uintptr) *sync.Pool {
	return &getPl[T](p).Pool
}

func getPl[T any](p uintptr) *pl {
	index := (p >> offset) & maxIndex
	var ss []poolUintptr
	var x *poolUintptr
	v := op.m[index].Load()
	if v != nil {
		ss = *v
		for i := range *v {
			x = &ss[i]
			if x.uintptr == p {
				return &x.singlePool
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
			x = &ss[i]
			if x.uintptr == p {
				lock.Unlock()
				return &x.singlePool
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
					return new(T)
				},
			},
		},
	}
	op.m[index].Store(&newSS)
	po := &newSS[l]
	lock.Unlock()
	return &po.singlePool
}

// Get a object from object pool with T
func Get[T any]() *T {
	return get[T](GetPtr[T]()).Get().(*T)
}

// Put a object to object pool with T
func Put[T any](t *T) {
	var a any = t
	if c, ok := a.(Clear); ok {
		c.Reset()
	}
	get[T](GetPtrAny(t)).Put(t)
}
