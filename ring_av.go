package engine

import (
	"container/ring"
	"context"
	"reflect"
	"runtime"
	"time"
)

type AVItem struct {
	Value   interface{}
	canRead bool
}
type AVRing struct {
	RingBuffer
	poll time.Duration
}

func (r *AVRing) Init(ctx context.Context, n int) *AVRing {
	r.Ring = ring.New(n)
	r.Context = ctx
	for x := r.Ring; x.Value == nil; x = x.Next() {
		x.Value = new(AVItem)
	}
	return r
}
func (rb AVRing) Clone() *AVRing {
	return &rb
}

func (r AVRing) SubRing(rr *ring.Ring) *AVRing {
	r.Ring = rr
	return &r
}
func (r *AVRing) Write(value interface{}) {
	last := r.Current()
	last.Value = value
	r.GetNext().canRead = false
	last.canRead = true
}

func (r *AVRing) Step() {
	last := r.Current()
	r.GetNext().canRead = false
	last.canRead = true
}

func (r *AVRing) wait() {
	if r.poll == 0 {
		runtime.Gosched()
	} else {
		time.Sleep(r.poll)
	}
}

func (r *AVRing) read() reflect.Value {
	current := r.Current()
	for r.Err() == nil && !current.canRead {
		r.wait()
	}
	return reflect.ValueOf(current.Value)
}

func (r *AVRing) nextRead() reflect.Value {
	r.MoveNext()
	return r.read()
}

func (r *AVRing) CurrentValue() interface{} {
	return r.Current().Value
}

func (r *AVRing) Current() *AVItem {
	return r.Ring.Value.(*AVItem)
}

func (r *AVRing) NextRead() interface{} {
	r.MoveNext()
	return r.Read()
}
func (r *AVRing) NextValue() interface{} {
	return r.Next().Value.(*AVItem).Value
}
func (r *AVRing) GetNext() *AVItem {
	r.MoveNext()
	return r.Current()
}
func (r *AVRing) Read() interface{} {
	current := r.Current()
	for r.Err() == nil && !current.canRead {
		r.wait()
	}
	return current.Value
}

// ReadLoop 循环读取，采用了反射机制，不适用高性能场景
// handler入参可以传入回调函数或者channel
func (r *AVRing) ReadLoop(handler interface{}) {
	switch t := reflect.ValueOf(handler); t.Kind() {
	case reflect.Chan:
		for v := r.read(); r.Err() == nil; v = r.nextRead() {
			t.Send(v)
		}
	case reflect.Func:
		for args := []reflect.Value{r.read()}; r.Err() == nil; args[0] = r.nextRead() {
			t.Call(args)
		}
	}
}

