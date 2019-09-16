package tctx2

import (
	"container/heap"
	"time"
)

type pair struct {
	obj  interface{}
	time time.Time
}

type Context struct {
	timer   *time.Timer
	reqs    *pairHeap
	reqChan chan pair
	C       chan interface{}
}

func (tc *Context) handleRequest(p pair) {
	heap.Push(tc.reqs, p)
	tc.timer.Reset(0)
}

func (tc *Context) handleTimeout(t time.Time) {
	for tc.reqs.Len() > 0 {
		p := (*tc.reqs)[0]
		m := p.time.Sub(t)
		if m <= 0 {
			tc.C <- p.obj
			heap.Pop(tc.reqs)
		} else {
			tc.timer.Reset(m)
			return
		}
	}
}

func (tc *Context) loop() {
	for {
		select {
		// new requests
		case d := <-tc.reqChan:
			tc.handleRequest(d)
		// for timeouts
		case t := <-tc.timer.C:
			tc.handleTimeout(t)
		}
	}
}

func (tc *Context) Add(obj interface{}, ext time.Duration) {
	tc.AddExact(obj, time.Now().Add(ext))
}

func (tc *Context) AddExact(obj interface{}, deadline time.Time) {
	tc.reqChan <- pair{obj, deadline}
}
