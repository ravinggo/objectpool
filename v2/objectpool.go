package v2

import (
	"math/bits"
	"sync"
	"unsafe"
)

//go:linkname memclrNoHeapPointers runtime.memclrNoHeapPointers
func memclrNoHeapPointers(ptr unsafe.Pointer, n uintptr)

// GoType copy from abi.Type
type GoType struct {
	Size_       uintptr
	PtrBytes    uintptr // number of (prefix) bytes in the type that can contain pointers
	Hash        uint32  // hash of type; avoids computation in hash tables
	TFlag       uint8   // extra type information flags
	Align_      uint8   // alignment of variable with this type
	FieldAlign_ uint8   // alignment of struct field with this type
	Kind_       uint8   // enumeration for C

	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	Equal func(unsafe.Pointer, unsafe.Pointer) bool
	// GCData stores the GC type data for the garbage collector.
	// Normally, GCData points to a bitmask that describes the
	// ptr/nonptr fields of the type. The bitmask will have at
	// least PtrBytes/ptrSize bits.
	// If the TFlagGCMaskOnDemand bit is set, GCData is instead a
	// **byte and the pointer to the bitmask is one dereference away.
	// The runtime will build the bitmask if needed.
	// (See runtime/type.go:getGCMask.)
	// Note: multiple types may have the same value of GCData,
	// including when TFlagGCMaskOnDemand is set. The types will, of course,
	// have the same pointer layout (but not necessarily the same size).
	GCData    *byte
	Str       int32 // string form
	PtrToThis int32 // type for pointer to this type, may be zero
}

// PtrGoType copy from abi.PtrType
type PtrGoType struct {
	GoType
	Elem *GoType
}

// EmptyInterface copy from abi.Type
type EmptyInterface struct {
	Type *GoType
	Data unsafe.Pointer
}

//go:linkname mallocgc runtime.mallocgc
func mallocgc(size uintptr, typ *GoType, needzero bool) unsafe.Pointer

func index(n uint32) uint32 {
	return uint32(bits.Len32(n - 1))
}

var pool [32]sync.Pool

func Get[T any]() *T {
	var t T
	size := unsafe.Sizeof(t)
	idx := index(uint32(size))
	if v := pool[idx].Get(); v != nil {
		return (*T)(v.(unsafe.Pointer))
	}
	var a any = t
	eface := *(*EmptyInterface)(unsafe.Pointer(&a))
	return (*T)(mallocgc(size, eface.Type, true))
}

func Put[T any](data *T) {
	var t T
	size := uint32(unsafe.Sizeof(t))
	idx := index(size)
	*data = t
	pool[idx].Put(unsafe.Pointer(data))
}

type Slice struct {
	data unsafe.Pointer
	len  int
	cap  int
}

func GetSlice[T any](c int) []T {
	if c <= 0 {
		return *(*[]T)(unsafe.Pointer(
			&Slice{
				len: 0,
				cap: 0,
			},
		))
	}
	idx := index(uint32(c))
	if v := pool[idx].Get(); v != nil {
		p := v.(unsafe.Pointer)
		s := &Slice{
			data: p,
			len:  0,
			cap:  1 << idx,
		}
		return *(*[]T)(unsafe.Pointer(s))
	}
	return make([]T, 0, 1<<idx)
}

func PutSlice[T any](data []T) {
	if len(data) > 0 {
		clear(data)
	}

	s := (*Slice)(unsafe.Pointer(&data))
	c := s.cap
	if c != 1<<index(uint32(c)) {
		return
	}
	pool[index(uint32(c))].Put(s.data)
}
