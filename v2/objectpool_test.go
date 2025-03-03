package v2

import (
	"sync"
	"testing"
	"unsafe"
)

func TestGetPutMap(t *testing.T) {
	wg := &sync.WaitGroup{}
	for goCount := 0; goCount < 100; goCount++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10000; i++ {
				hm := Get[Struct]()
				hm.U8++
				Put(hm)
				//
				s := GetSlice[byte](int(unsafe.Sizeof(Struct{})))
				s = append(s, 1, 2, 3, 43, 54, 6, 6, 7, 3)
				PutSlice(s)
			}
		}()
	}
	wg.Wait()
}

type Struct struct {
	byte byte
	I8   int8
	U8   uint8
	I16  int16
	I32  int32
	// I64  int64
	// F32  float32
	// F64  float64
	// S    string
	// ss   []string
}

func BenchmarkGetPut(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Put(Get[Struct]())
	}
}

var sp = sync.Pool{
	New: func() interface{} {
		return &Struct{}
	},
}

func BenchmarkSyncPoolGetPut(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := sp.Get().(*Struct)
		sp.Put(d)
	}
}

func BenchmarkGetPutSlice(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PutSlice(GetSlice[Struct](128))
	}
}
