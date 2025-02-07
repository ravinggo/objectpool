package objectpool

import (
	"math"
	"math/bits"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	maxIndex    = math.MaxUint16 - 1
	otherMinCap = 16
	byteMinCap  = 128
	KindMask    = (1 << 5) - 1
	offset      = 5
)

func GetTypePool[T any]() *sync.Pool {
	return get[T](GetPtr[T]())
}

func GetSliceTypePool[T any]() *SlicePool {
	return getSlice[T](GetPtr[Slice[T]]())
}

func GetPtrSliceTypePool[T any]() *SlicePool {
	return getSlice[T](GetPtr[[]T]())
}

func GetMapTypePool[K comparable, V any]() *sync.Pool {
	return getMap[K, V](GetMapPtr[K, V]())
}

// Clear is an interface that can be implemented by a struct to reset its value
// Just for the convenience of objectpool
type Clear interface {
	Reset()
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

// PtrType copy from abi.PtrType
type PtrType struct {
	Type
	Elem *Type
}

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

type mapPtr[K comparable, V any] struct {
	x map[K]V
}

// GetMapPtr get type pointer of map[K]V
func GetMapPtr[K comparable, V any]() uintptr {
	var a any = (*mapPtr[K, V])(nil)
	t := *(**Type)(unsafe.Pointer(&a))
	return (uintptr)(unsafe.Pointer(t))
}

var (
	op       = objectPool{}
	bytesPtr = func() uintptr {
		return GetPtr[Slice[byte]]()
	}()
	bytesPool = getSlice[byte](bytesPtr)
)

type spl struct {
	SlicePool
	PutPointer
}
type pl struct {
	sync.Pool
	PutPointer
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
			PutPointer: putKA[T],
		},
	}
	op.m[index].Store(&newSS)
	po := &newSS[l]
	lock.Unlock()
	return &po.singlePool
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
					return map[K]V{}
				},
			},
			PutPointer: putKAMap[K, V],
		},
	}
	op.m[index].Store(&newSS)
	po := &newSS[l]
	lock.Unlock()
	return &po.singlePool
}

func getSlice[T any](p uintptr) *SlicePool {
	return &getSliceSpl[T](p).SlicePool
}

func getSliceSpl[T any](p uintptr) *spl {
	index := (p >> offset) & maxIndex
	var ss []poolUintptr
	v := op.m[index].Load()
	if v != nil {
		ss = *v
		for i := range ss {
			if ss[i].uintptr == p {
				return &ss[i].slicePool
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
			if ss[i].uintptr == p {
				lock.Unlock()
				return &ss[i].slicePool
			}
		}
	}

	l := len(ss)
	newSS := make([]poolUintptr, l+1)
	copy(newSS, ss)
	newSS[l] = poolUintptr{
		uintptr: p,
		slicePool: spl{
			PutPointer: putKASlice[T],
		},
	}
	op.m[index].Store(&newSS)
	po := &newSS[l]
	lock.Unlock()
	return &po.slicePool
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

// Slice  is a slice object pool for T
// []T put sync.Pool is invalid
type Slice[T any] struct {
	Data []T
}

type SlicePool struct {
	pools [32]sync.Pool
}

func index(n uint32) uint32 {
	return uint32(bits.Len32(n - 1))
}

func getSlicePool[T any](s *SlicePool, cap int, minCap int) *Slice[T] {
	if cap > math.MaxInt32 {
		return &Slice[T]{Data: make([]T, cap)}
	}

	if cap < minCap { // 小内存分配太零散了。128字节起步，复用率比较高
		cap = minCap
	}

	idx := index(uint32(cap))
	if v := s.pools[idx].Get(); v != nil {
		bp := v.(*Slice[T])
		return bp
	}
	return &Slice[T]{Data: make([]T, 0, 1<<idx)}
}

func getPtrSlicePool[T any](s *SlicePool, cap int, minCap int) *[]T {
	if cap > math.MaxInt32 {
		ret := make([]T, cap)
		return &ret
	}

	if cap < minCap { // 小内存分配太零散了。128字节起步，复用率比较高
		cap = minCap
	}

	idx := index(uint32(cap))
	if v := s.pools[idx].Get(); v != nil {
		bp := v.(*[]T)
		return bp
	}
	ret := make([]T, 0, 1<<idx)
	return &ret
}

func putSlicePool[T any](s *SlicePool, t *Slice[T]) {
	t.Data = t.Data[:0]
	c := cap(t.Data)
	idx := index(uint32(c))
	if c != 1<<idx { // 不是Get获取的[]byte，放在前一个索引的Pool里面
		idx--
	}
	s.pools[idx].Put(t)
}

func putPtrSlicePool[T any](s *SlicePool, t *[]T) {
	*t = (*t)[:0]
	c := cap(*t)
	idx := index(uint32(c))
	if c != 1<<idx { // 不是Get获取的[]byte，放在前一个索引的Pool里面
		idx--
	}
	s.pools[idx].Put(t)
}

// GetSlice get a slice from object pool with T,len() == 0
func GetSlice[T any](cap int) *Slice[T] {
	typPtr := GetPtr[Slice[T]]()
	s := getSlice[T](typPtr)
	var minCap int
	if typPtr != bytesPtr {
		minCap = otherMinCap
	} else {
		minCap = byteMinCap
	}
	return getSlicePool[T](s, cap, minCap)
}

func GetPtrSlice[T any](cap int) *[]T {
	typPtr := GetPtr[[]T]()
	s := getSlice[T](typPtr)
	var minCap int
	if typPtr != bytesPtr {
		minCap = otherMinCap
	} else {
		minCap = byteMinCap
	}
	return getPtrSlicePool[T](s, cap, minCap)
}

func GetSlice2[T any](s *SlicePool, cap int) *Slice[T] {
	typPtr := GetPtr[Slice[T]]()
	var minCap int
	if typPtr != bytesPtr {
		minCap = otherMinCap
	} else {
		minCap = byteMinCap
	}

	return getSlicePool[T](s, cap, minCap)
}

func GetPtrSlice2[T any](s *SlicePool, cap int) *[]T {
	typPtr := GetPtr[[]T]()
	var minCap int
	if typPtr != bytesPtr {
		minCap = otherMinCap
	} else {
		minCap = byteMinCap
	}
	return getPtrSlicePool[T](s, cap, minCap)
}

// GetSliceForSize  get a slice from object pool with T and size, len() == size
func GetSliceForSize[T any](size int) *Slice[T] {
	s := GetSlice[T](size)
	s.Data = s.Data[:size]
	return s
}

func GetPtrSliceForSize[T any](size int) *[]T {
	s := GetPtrSlice[T](size)
	*s = (*s)[:size]
	return s
}

func GetSliceForSize2[T any](sp *SlicePool, size int) *Slice[T] {
	s := GetSlice2[T](sp, size)
	s.Data = s.Data[:size]
	return s
}

func GetPtrSliceForSize2[T any](sp *SlicePool, size int) *[]T {
	s := GetPtrSlice2[T](sp, size)
	*s = (*s)[:size]
	return s
}

// PutSlice put a slice to object pool with T
func PutSlice[T any](t *Slice[T]) {
	if cap(t.Data) > math.MaxInt32 {
		return
	}
	typPtr := GetPtr[Slice[T]]()
	if typPtr != bytesPtr {
		if cap(t.Data) < otherMinCap {
			return
		}
	} else {
		if cap(t.Data) < byteMinCap {
			return
		}
	}

	s := getSlice[T](typPtr)
	putSlicePool(s, t)
}

func PutSliceClear[T any](t *Slice[T]) {
	if cap(t.Data) > math.MaxInt32 {
		return
	}
	typPtr := GetPtr[Slice[T]]()
	if typPtr != bytesPtr {
		if cap(t.Data) < otherMinCap {
			return
		}
		clear(t.Data)
	} else {
		if cap(t.Data) < byteMinCap {
			return
		}
	}

	s := getSlice[T](typPtr)
	putSlicePool(s, t)
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

// Bytes is a slice object pool for byte
type Bytes Slice[byte]

func GetBytes(cap int) *Bytes {
	b := (*Bytes)(getSlicePool[byte](bytesPool, cap, byteMinCap))
	b.Data = b.Data[:0]
	return b
}

func PutBytes(b *Bytes) {
	PutSlice((*Slice[byte])(b))
}

func (b *Bytes) WriteString(s string) {
	b.Data = append(b.Data, s...)
}

func (b *Bytes) WriteBytes(c ...byte) {
	b.Data = append(b.Data, c...)
}

func (b *Bytes) String() string {
	return unsafe.String(unsafe.SliceData(b.Data), len(b.Data))
}

func (b *Bytes) WriteInt(i int64) {
	b.Data = strconv.AppendInt(b.Data, i, 10)
}

func (b *Bytes) WriteUint(i uint64) {
	b.Data = strconv.AppendUint(b.Data, i, 10)
}

func (b *Bytes) WriteFloat(f float64) {
	b.Data = strconv.AppendFloat(b.Data, f, 'g', -1, 64)
}

func (b *Bytes) WriteBool(v bool) {
	if v {
		b.WriteBytes('T')
	} else {
		b.WriteBytes('F')
	}
}

func (b *Bytes) Len() int {
	return len(b.Data)
}

func (b *Bytes) Cap() int {
	return cap(b.Data)
}

func (b *Bytes) Bytes() []byte {
	return b.Data
}

func (b *Bytes) Reset() {
	b.Data = b.Data[:0]
}

type KeepAliveElem struct {
	put  uintptr
	pool uintptr
	// unsafe.Pointer for keepalive
	data unsafe.Pointer
}

type KeepAlive struct {
	elems []KeepAliveElem
}

func NewKeepAlive(cap int) *KeepAlive {
	return &KeepAlive{make([]KeepAliveElem, 0, cap)}
}

func (ka *KeepAlive) Reset() {
	for _, e := range ka.elems {
		f := *(*PutPointer)(unsafe.Pointer(e.put))
		f(e.pool, e.data)
	}

	clear(ka.elems)
	ka.elems = ka.elems[:0]
}

func GetKA[T any](ka *KeepAlive) *T {
	pl := getPl[T](GetPtr[T]())
	v := pl.Pool.Get().(*T)
	ka.elems = append(
		ka.elems, KeepAliveElem{
			put:  uintptr(unsafe.Pointer(&pl.PutPointer)),
			pool: uintptr(unsafe.Pointer(&pl.Pool)),
			data: unsafe.Pointer(v),
		},
	)
	return v
}

// GetKASlice get a slice from object pool with T and cap
// GetKASlice == make([]T,0,cap)
func GetKASlice[T any](ka *KeepAlive, cap int) *Slice[T] {
	typPtr := GetPtr[Slice[T]]()
	spl := getSliceSpl[T](typPtr)
	var minCap int
	if typPtr != bytesPtr {
		minCap = otherMinCap
	} else {
		minCap = byteMinCap
	}
	v := getSlicePool[T](&spl.SlicePool, cap, minCap)
	ka.elems = append(
		ka.elems, KeepAliveElem{
			put:  uintptr(unsafe.Pointer(&spl.PutPointer)),
			pool: uintptr(unsafe.Pointer(&spl.SlicePool)),
			data: unsafe.Pointer(v),
		},
	)
	return v
}

// GetKASliceSize get a slice from object pool with T and size, len() == size
// GetKASliceSize == make([]T,size,cap)
func GetKASliceSize[T any](ka *KeepAlive, size, cap int) *Slice[T] {
	if cap < size {
		cap = size
	}
	ret := GetKASlice[T](ka, cap)
	ret.Data = ret.Data[:size]
	return ret
}

func GetKAMap[K comparable, V any](ka *KeepAlive) map[K]V {
	pl := getMapPl[K, V](GetMapPtr[K, V]())
	v := pl.Pool.Get().(map[K]V)
	ka.elems = append(
		ka.elems, KeepAliveElem{
			put:  uintptr(unsafe.Pointer(&pl.PutPointer)),
			pool: uintptr(unsafe.Pointer(&pl.Pool)),

			// why? map is *runtime.hmap,data is *hmap
			// (unsafe.Pointer(&v)) get v(stack) address
			// *(*uintptr)(unsafe.Pointer(&v)) get *runtime.hmap to unsafe.Pointer keepalive
			data: unsafe.Pointer(*(*uintptr)(unsafe.Pointer(&v))),
		},
	)
	return v
}

type PutPointer func(uintptr, unsafe.Pointer)

func putKA[T any](pool uintptr, p unsafe.Pointer) {
	var a any = (*T)(p)
	if c, ok := a.(Clear); ok {
		c.Reset()
	}
	(*sync.Pool)(unsafe.Pointer(pool)).Put(a)
}

func putKASlice[T any](pool uintptr, p unsafe.Pointer) {
	t := (*Slice[T])(p)
	if cap(t.Data) > math.MaxInt32 {
		return
	}
	typPtr := GetPtr[Slice[T]]()
	if typPtr != bytesPtr {
		if cap(t.Data) < otherMinCap {
			return
		}
		clear(t.Data)
	} else {
		if cap(t.Data) < byteMinCap {
			return
		}
	}
	putSlicePool((*SlicePool)(unsafe.Pointer(pool)), t)
}

func putKAMap[K comparable, V any](pool uintptr, p unsafe.Pointer) {
	// p is *runtime.hmap
	x := (uintptr)(p)
	// get x(stack) address: **runtime.hmap
	y := unsafe.Pointer(&x)
	// get *(**runtime.hmap)(y) to unsafe.Pointer
	m := *(*map[K]V)(y)
	clear(m)
	(*sync.Pool)(unsafe.Pointer(pool)).Put(m)
}
