package util

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
)

// TODO(remy): comment me
var GlobalStringSlicePool = NewStringSlicePool()

type StringSlicePool struct {
	sync.Pool
}

// TODO(remy): comment me
func NewStringSlicePool() *StringSlicePool {
	return &StringSlicePool{
		Pool: sync.Pool{
			New: func() interface{} {
				// TODO(remy): configurable default size?
				return NewStringSlice(10)
			},
		},
	}
}

func (s *StringSlicePool) Get() *StringSlice {
	//fmt.Println("Get")
	rv := s.Pool.Get().(*StringSlice)
	rv.Reset()
	return rv
}

func (s *StringSlicePool) GetWithValues(values []string) *StringSlice {
	//fmt.Printf("GetWithValues(%s)", values)
	rv := s.Pool.Get().(*StringSlice)
	rv.Reset()
	rv.AppendMany(values)
	return rv
}

func (s *StringSlicePool) Put(sl *StringSlice) {
	//fmt.Println("Put")
	s.Pool.Put(sl)
}

type StringSlice struct {
	array []string
	// capacity is the total size that can contain the backing array.
	// It's also the value that would be the equivalent of cap(slice)
	capacity uint
	// offset is the current position we're at in the array.
	// It's also the value that would be the equivalent of len(slice)
	offset uint
}

var EmptyStringSlice = &StringSlice{
	array:    nil,
	capacity: 0,
	offset:   0,
}

func NewStringSlice(capacity uint) *StringSlice {
	return &StringSlice{
		array:    make([]string, capacity),
		capacity: capacity,
		offset:   0,
	}
}

func (d *StringSlice) Append(value string) {
	if d.offset+1 > d.capacity {
		d.array = append(d.array, value) // dynamically grow in this case
		d.capacity++
		d.offset++
		//fmt.Printf("%p Append(%s) (grow)\n", d, value)
		return
	}
	d.array[d.offset] = value
	d.offset++
	//fmt.Printf("%p Append(%s)\n", d, value)
	return
}

func (d *StringSlice) AppendMany(values []string) {
	for _, value := range values {
		d.Append(value)
	}
}

// Copy copies the current StringSlice and returns a new one
// which comes from the Pool (the caller is responsible of
// pushing back the new one to the Pool)
func (d *StringSlice) Copy() *StringSlice {
	rv := GlobalStringSlicePool.Get()
	rv.AppendMany(d.Slice())
	return rv
}

func (d *StringSlice) Slice() []string {
	// TODO(remy): is this an actual fix?
	if d == nil {
		return []string{}
	}
	if d.array == nil {
		return []string{}
	}
	return d.array[:d.offset]
}

func (d *StringSlice) SliceCopy() []string {
	rv := make([]string, d.offset)
	copy(rv, d.array[:d.offset])
	return rv
}

func (d *StringSlice) Get(index uint) string {
	return d.array[index]
}

func (d *StringSlice) Set(index uint, v string) {
	//fmt.Printf("%p Set(%d, %s)\n", d, index, v)
	d.array[index] = v
}

func (d *StringSlice) Resize(size uint) {
	//fmt.Printf("%p Resize(%d)\n", d, size)
	// here, we don't want to change the capacity because a later use
	// of the string slice may need that extra memory.
	d.offset = size
}

func (d *StringSlice) Reset() {
	//fmt.Printf("%p Reset()\n", d)
	// TODO(remy): maybe we can add a strategy resetting the backing array
	// here if it is way too big?
	d.offset = 0
}

func (d *StringSlice) Len() uint {
	return d.offset
}

func (d *StringSlice) Cap() uint {
	return d.capacity
}

// TODO(remy): comment me
func (d *StringSlice) MarshalJSON() ([]byte, error) {
	buff := bytes.NewBuffer(nil)
	buff.WriteString("[")
	first := true
	// ignore device field
	for _, tag := range d.Slice() {
		if strings.HasPrefix(tag, "device:") {
			continue
		}
		if !first {
			buff.WriteByte(',')
		}
		buff.WriteString(fmt.Sprintf("\"%s\"", tag))
		first = false
	}
	buff.WriteString("]")
	return buff.Bytes(), nil
}
