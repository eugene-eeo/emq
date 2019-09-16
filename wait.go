package main

import "time"

type Waiters struct {
	head *Waiter
	tail *Waiter
}

func (ws *Waiters) AddWaiter(w *Waiter) {
	if ws.tail == nil {
		ws.head = w
		ws.tail = w
	} else {
		w.Prev = ws.tail
		ws.tail.Next = w
		ws.tail = w
	}
}

func (ws *Waiters) Remove(w *Waiter) (*Waiter, *Waiter) {
	prev := w.Prev
	next := w.Next
	if prev != nil {
		prev.Next = next
	} else {
		ws.head = next
	}
	if next != nil {
		next.Prev = prev
	} else {
		ws.tail = prev
	}
	// Clear for GC
	w.Next = nil
	w.Prev = nil
	return prev, next
}

func (ws *Waiters) Update(queues map[string]*Queue) {
	curr := ws.head
	for curr != nil {
		ready := curr.IsReady(queues)
		if ready {
			_, next := ws.Remove(curr)
			curr.Consume(queues)
			curr.EmitReady()
			curr = next
		} else {
			curr = curr.Next
		}
	}
}

type WaitConfigJson struct {
	NoWait  bool     `json:"no_wait"`
	Queues  []string `json:"queues"`
	Timeout int      `json:"timeout"` // Timeout in seconds
}

type Waiter struct {
	NoWait   bool
	Queues   []string
	Tasks    []*Task
	Ready    chan bool
	Prev     *Waiter
	Next     *Waiter
	Deadline time.Time
}

func NewWaiterFromConfig(wc *WaitConfigJson) *Waiter {
	return &Waiter{
		NoWait: wc.NoWait,
		Queues: wc.Queues,
		Tasks:  make([]*Task, len(wc.Queues)),
		Ready:  make(chan bool, 1),
	}
}

func (w *Waiter) EmitReady() {
	w.Ready <- true
	close(w.Ready)
}

func (w *Waiter) IsReady(queues map[string]*Queue) bool {
	if w.NoWait {
		return true
	}
	counts := map[string]int{}
	for _, x := range w.Queues {
		counts[x]++
	}
	for x, n := range counts {
		q := queues[x]
		if q == nil || q.count < n {
			return false
		}
	}
	return true
}

func (w *Waiter) Consume(queues map[string]*Queue) {
	for i, name := range w.Queues {
		q := queues[name]
		if q != nil {
			w.Tasks[i] = q.Dequeue()
		}
	}
}
