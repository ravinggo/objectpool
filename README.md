# objectpool
Generic Object Pool
## Usage
```go

type Struct struct {
    Name string
}

// a = &Struct{}
a:=objectpool.Get[Struct]()
defer objectpool.Put(a)

// len(s) == 0,  cap(s) == 16
// s = &Slice{Data: make([]Struct, 0, 16)}
s:=objectpool.GetSlice[Struct](10)
defer objectpool.PutSlice(s)

// len(s) == 0,  cap(s) == 128
s1 =objectpool.GetSlice[Struct](127)
defer objectpool.PutSlice(s1)

// m = map[int]int{}
m:= objectpool.GetMap[int,int]()
defer objectpool.PutMap(m)

ka := NewKeepAlive(3)
defer ka.Reset()

a1:=GetKA[Struct](ka)
a2:=GetKASlice[Struct](ka, 12)
a3:=GetKASliceSize[Struct](ka,16,32)
s3:=GetKAMap[int, *Struct](ka)

```

## benchmark

```go

goos: windows
goarch: amd64
pkg: github.com/ravinggo/objectpool
cpu: AMD Ryzen 5 5600 6-Core Processor
BenchmarkGetPut
BenchmarkGetPut-12              	73385068	        16.30 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetSlicePutSlice
BenchmarkGetSlicePutSlice-12    	63563024	        19.35 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetMapPutMap
BenchmarkGetMapPutMap-12        	71844670	        16.77 ns/op	       0 B/op	       0 allocs/op
BenchmarkPool
BenchmarkPool-12                	121695349	         9.852 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetKA
BenchmarkGetKA-12               	19810184	        60.24 ns/op	       0 B/op	       0 allocs/op
BenchmarkMallocgc
BenchmarkMallocgc-12            	14704152	        85.72 ns/op	     152 B/op	       3 allocs/op
PASS

```

## We welcome suggestions for optimization. We really can't optimize anymore.