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

```

## benchmark

```go
goos: linux
goarch: amd64
pkg: github.com/ravinggo/objectpool
cpu: Intel(R) Core(TM) i9-14900KF
BenchmarkGetPut-32                      112631061               10.87 ns/op            0 B/op          0 allocs/op
BenchmarkGetSlicePutSlice-32            100000000               10.88 ns/op            0 B/op          0 allocs/op
BenchmarkGetMapPutMap-32                100000000               10.02 ns/op            0 B/op          0 allocs/op
BenchmarkPool-32                        245529364                5.267 ns/op           0 B/op          0 allocs/op
```

## We welcome suggestions for optimization. We really can't optimize anymore.