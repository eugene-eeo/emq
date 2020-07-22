package main

import "time"

type WaitSpecConfig struct {
	Queues  []string `json:"queues"`
	Timeout uint32   `json:"timeout"`
}

type WaitDone struct {
	now   time.Time
	tasks []*Task
}

type WaitSpec struct {
	Queues  []string
	Timeout time.Duration
	ready   chan WaitDone
	// linked list
	prev *WaitSpec
	next *WaitSpec
	q    *Waiters
}

func (wsc WaitSpecConfig) ToWaitSpec() WaitSpec {
	return WaitSpec{
		Queues:  wsc.Queues,
		Timeout: time.Duration(wsc.Timeout) * time.Millisecond,
		ready:   make(chan WaitDone),
	}
}

func (w WaitSpec) Take(mq *MQ, now time.Time) []*Task {
	tasks := make([]*Task, len(w.Queues))
	heads := map[*Queue]*Task{}
	for i, qn := range w.Queues {
		q := mq.Queues[qn]
		if q == nil {
			continue
		}
		if _, ok := heads[q]; !ok {
			heads[q] = q.Head()
		}
		tasks[i], heads[q] = q.NextUndispatched(heads[q], now)
	}
	return tasks
}

func (w WaitSpec) Ready(mq *MQ, now time.Time) ([]*Task, bool) {
	tasks := make([]*Task, len(w.Queues))
	heads := map[*Queue]*Task{}
	for i, qn := range w.Queues {
		q := mq.Queues[qn]
		if q == nil {
			return nil, false
		}
		if _, ok := heads[q]; !ok {
			heads[q] = q.Head()
		}
		t, next := q.NextUndispatched(heads[q], now)
		if t == nil {
			return nil, false
		}
		tasks[i] = t
		heads[q] = next
	}
	return tasks, true
}

func (w WaitSpec) Next() *WaitSpec {
	if w.next == &w.q.root {
		return nil
	}
	return w.next
}

type Waiters struct {
	root WaitSpec
}

func NewWaiters() *Waiters {
	w := &Waiters{}
	w.root.q = w
	w.root.prev = &w.root
	w.root.next = &w.root
	return w
}

func (ws *Waiters) Remove(w *WaitSpec) {
	w.prev.next = w.next
	w.next.prev = w.prev
	w.next = nil
	w.prev = nil
	w.q = nil
}

func (ws *Waiters) Append(w *WaitSpec) {
	w.q = ws
	at := ws.root.prev
	w.next = at.next
	w.prev = at
	at.next.prev = w
	at.next = w
}

func (ws *Waiters) Empty() bool {
	return ws.root.next == &ws.root
}

func (ws *Waiters) Head() *WaitSpec {
	if ws.Empty() {
		return nil
	}
	return ws.root.next
}
