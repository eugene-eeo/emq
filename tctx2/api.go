package tctx2

import "time"

func NewContext() *Context {
	tc := &Context{
		timer:   time.NewTimer(0),
		reqs:    &pairHeap{},
		reqChan: make(chan pair),
		C:       make(chan interface{}),
	}
	go tc.loop()
	return tc
}
