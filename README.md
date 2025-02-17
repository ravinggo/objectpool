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
// s = make([]Struct, 0, 16)
s:=objectpool.GetSlice[Struct](10)
defer objectpool.PutSlice(s)

// len(s) == 0,  cap(s) == 128
s1 =objectpool.GetSlice[Struct](127)
defer objectpool.PutSlice(s1)

// m = map[int]int{}
m:= objectpool.GetMap[int,int]()
defer objectpool.PutMap(m)

// Get Sync.Pool of Type
objectpool.GetTypePool[T]() == &sync.Pool{ New: func() any { return new(T) }}
objectpool.GetSliceTypePool[T]() == &sync.Pool{ New: func() any { return make([]T, 0, cap) }}
objectpool.GetTypePool[K,T]() == &sync.Pool{ New: func() any { return map[K]V }}
```

## benchmark

```go
goos: windows
goarch: amd64
pkg: github.com/ravinggo/objectpool
cpu: AMD Ryzen 5 5600 6-Core Processor
BenchmarkGetPut
BenchmarkGetPut-12              	67208440	        18.62 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetSlicePutSlice
BenchmarkGetSlicePutSlice-12    	57129798	        21.40 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetMapPutMap
BenchmarkGetMapPutMap-12        	70459749	        17.49 ns/op	       0 B/op	       0 allocs/op
BenchmarkMallocgc
BenchmarkMallocgc-12            	54508990	        21.42 ns/op	       8 B/op	       1 allocs/op
BenchmarkMallocgcSlice
BenchmarkMallocgcSlice-12       	37174836	        36.18 ns/op	     128 B/op	       1 allocs/op
BenchmarkMallocgcMap
BenchmarkMallocgcMap-12         	 7313108	       161.6 ns/op	     688 B/op	       2 allocs/op
BenchmarkSyncPool
BenchmarkSyncPool-12            	100000000	        10.61 ns/op	       0 B/op	       0 allocs/op
BenchmarkSyncPoolSlice
BenchmarkSyncPoolSlice-12       	100000000	        10.53 ns/op	       0 B/op	       0 allocs/op
BenchmarkSyncPoolMap
BenchmarkSyncPoolMap-12         	100000000	        10.77 ns/op	       0 B/op	       0 allocs/op
PASS
```

## We welcome suggestions for optimization. We really can't optimize anymore.