package gorx

import (
	"sync"
	"time"
)

const PROMISE_EXPIRED = "__expired__"

type Promise struct {
	mutex       *sync.Mutex
	callbacks   []func(v interface{})
	resolveOnce *sync.Once
}

func NewPromiseWithTimeout(timeout time.Duration) *Promise {
	ret := &Promise{callbacks: make([]func(v interface{}), 0),
		mutex: &sync.Mutex{}, resolveOnce: &sync.Once{}}
	go func() {
		time.Sleep(timeout)
		ret.Resolve(PROMISE_EXPIRED)
	}()
	return ret
}

func (p *Promise) Resolve(val interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.resolveOnce.Do(func() {
		for _, cb := range p.callbacks {
			go cb(val)
		}
	})
}

func (p *Promise) Then(cb func(v interface{})) *Promise {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.callbacks = append(p.callbacks, cb)
	return p
}
