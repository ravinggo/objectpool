package objectpool

import (
	_ "net/http/pprof"
	"reflect"
	"sync"
	"testing"
	"unsafe"
)

type Struct struct {
	A unsafe.Pointer
}

type Struct1 struct {
	a int
	b string
}

type Struct2 struct {
	a int
	b string
}

func TestGet(t *testing.T) {
	GetPtr[*Struct1]()

	s1 := Get[Struct1]()
	Put(s1)
	s2 := Get[Struct1]()
	if unsafe.Pointer(s1) != unsafe.Pointer(s2) {
		t.Fatal("s1 not equal s2")
	}
	Put(s2)

	s3 := Get[Struct2]()
	if unsafe.Pointer(s2) == unsafe.Pointer(s3) {
		t.Fatal("s2 equal s3")
	}

	bs := GetSlice[byte](0)
	if cap(bs) != byteMinCap {
		t.Fatalf("GetSlice[byte](0) cap is not %d", byteMinCap)
	}
	bs = GetSliceForSize[byte](129)
	if cap(bs) != 256 {
		t.Fatalf("GetSlice[byte](129) cap is not %d", 256)
	}
	if len(bs) != 129 {
		t.Fatalf("GetSlice[byte](129) len is not %d", 129)
	}
	PutSlice(bs)
	bss := GetSlice[Struct1](0)
	if cap(bss) != otherMinCap {
		t.Fatalf("GetSlice[Struct1](0)cap is not %d", otherMinCap)
	}
	PutSlice(bss)
	bss = GetSlice[Struct1](0)
	if cap(bss) != otherMinCap {
		t.Fatalf("GetSlice[Struct1](0)cap is not %d", otherMinCap)
	}
	PutSlice(bss)
	bss1 := GetSliceForSize[*Struct1](17)
	if cap(bss1) != 32 {
		t.Fatalf("GetSlice[Struct1](0)cap is not %d", 32)
	}
	if len(bss1) != 17 {
		t.Fatalf("GetSlice[Struct1](17) len is not %d", 17)
	}
	PutSlice(bss1)

	m := GetMap[int, int]()
	m[1] = 1
	PutMap(m)
	m = GetMap[int, int]()
	if len(m) != 0 {
		t.Fatalf("GetMap[int, int]() len is not %d", 0)
	}
	m1 := GetMap[string, string]()
	m1["1"] = "1"
	PutMap(m)
	m1 = GetMap[string, string]()
	if len(m) != 0 {
		t.Fatalf("GetMap[string, string]() len is not %d", 0)
	}
	p1 := GetPtr[Struct1]()
	p2 := GetPtrAny(&Struct1{})
	p3 := GetPtrAny(Struct1{})
	if p1 != p2 {
		t.Fatalf("GetPtr[Struct1] not equal GetPtrAny(&Struct1{})")
	}
	if p1 != p3 {
		t.Fatalf("GetPtr[Struct1] not equal GetPtrAny(Struct1{})")
	}

	b := GetBytes(127)
	if len(b) != 0 {
		t.Fatalf("GetBytes(127) len is not %d", 0)
	}

	if cap(b) != 128 {
		t.Fatalf("GetBytes(127) len is not %d", 128)
	}
	b.WriteBytes('1', '2', '3')
	if b.String() != "123" {
		t.Fatalf("GetBytes(127) Data is not %s", "123")
	}
	PutBytes(b)
	ptrCheck := map[uintptr]struct{}{}
	addSet(ptrCheck, GetMapPtr[int, int](), GetMapPtr[string, int]())
	addSet(ptrCheck, GetMapPtr[string, string](), GetMapPtr[int, string](), GetMapPtr[int, Struct1]())
	addSet(ptrCheck, GetMapPtr[Struct1, string](), GetMapPtr[string, Struct1](), GetMapPtr[Struct1, Struct1]())
	addSet(ptrCheck, GetPtr[int](), GetPtr[string](), GetPtr[Struct1](), GetMapPtr[Struct1, *Struct1]())
	addSet(ptrCheck, GetPtr[Struct](), GetPtr[*Struct](), GetPtr[*Struct1](), GetPtr[sliceType[int]]())
	addSet(ptrCheck, GetPtr[sliceType[int64]](), GetPtr[int64](), GetPtr[*int64](), GetPtr[sliceType[byte]]())
	addSet(ptrCheck, GetPtr[sliceType[*int64]](), GetPtr[uint64](), GetPtr[*uint64](), GetPtr[sliceType[rune]]())
	addSet(ptrCheck, GetPtr[[]int64]())
	if GetMapPtr[int, int]() == GetPtr[map[int]int]() {
		t.Fatalf("GetMapPtr[int, int]() == GetPtr[map[int]int]() ,must not equal")
	}

	mPtr := Get[map[string]string]()
	if reflect.TypeOf(mPtr) != reflect.TypeOf((*map[string]string)(nil)) {
		t.Fatalf("reflect.TypeOf(Get[map[string]string]()) != *map[string]string")
	}
	out := GetSlice[*Struct1](1)
	to := reflect.TypeOf(out)
	if to.Kind() != reflect.Slice {
		t.Errorf("GetSlice[*Struct1](1) is not slice")
	}
	if to.Elem().Kind() != reflect.Ptr {
		t.Errorf("GetSlice[*Struct1](1).Data Type is not ptr")
	}
}

func addSet(m map[uintptr]struct{}, keys ...uintptr) {
	for _, key := range keys {
		if _, ok := m[key]; ok {
			panic("key already exist")
		}
		m[key] = struct{}{}
	}
}

func BenchmarkGetPut(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := Get[Struct]()
		y := x
		Put(y)
	}
}

func BenchmarkGetSlicePutSlice(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := GetSlice[byte](127)
		y := x
		PutSlice(y)
	}
}

func BenchmarkGetMapPutMap(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := GetMap[int, Struct]()
		y := x
		PutMap(y)
	}
}

var a1 *Struct
var a2 []byte
var a3 map[int]*Struct

func BenchmarkMallocgc(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a1 = &Struct{}
	}
}

func BenchmarkMallocgcSlice(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a2 = make([]byte, 128)
	}
}

func BenchmarkMallocgcMap(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a3 = make(map[int]*Struct, 16)
	}
}

var p1 = sync.Pool{
	New: func() interface{} {
		return &Struct{}
	},
}

func BenchmarkSyncPool(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := p1.Get()
		y := x
		p1.Put(y)
	}
}

var p2 = sync.Pool{
	New: func() interface{} {
		ret := make([]int, 16)
		return &ret
	},
}

func BenchmarkSyncPoolSlice(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := p2.Get()
		y := x
		p2.Put(y)
	}
}

var p3 = sync.Pool{
	New: func() interface{} {
		ret := make(map[int]Struct, 16)
		return ret
	},
}

func BenchmarkSyncPoolMap(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := p3.Get().(map[int]Struct)
		y := x
		p3.Put(y)
	}
}
