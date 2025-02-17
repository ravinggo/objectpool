package objectpool

import (
	"unsafe"
)

// Clear is an interface that can be implemented by a struct to reset its value
// Just for the convenience of objectpool
type Clear interface {
	Reset()
}

// PtrType copy from abi.PtrType
type PtrType struct {
	Type
	Elem *Type
}

// Type copy from abi.Type
type Type struct {
	Size_       uintptr
	PtrBytes    uintptr
	Hash        uint32
	TFlag       uint8
	Align_      uint8
	FieldAlign_ uint8
	Kind_       uint8
	Equal       func(unsafe.Pointer, unsafe.Pointer) bool
	GCData      *byte
	Str         int32
	PtrToThis   int32
}

// MapType copy from abi.MapType
type MapType struct {
	Type
	Key    *Type
	Elem   *Type
	Bucket *Type // internal type representing a hash bucket
	// function for hashing keys (ptr to key, seed) -> hash
	Hasher     func(unsafe.Pointer, uintptr) uintptr
	KeySize    uint8  // size of key slot
	ValueSize  uint8  // size of elem slot
	BucketSize uint16 // size of bucket
	Flags      uint32
}
