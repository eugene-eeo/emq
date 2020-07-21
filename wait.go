package main

import "time"

type WaitSpecConfig struct {
	Queues  []string `json:"queues"`
	Timeout int64    `json:"timeout"`
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
}

func (wsc WaitSpecConfig) ToWaitSpec() WaitSpec {
	return WaitSpec{
		Queues:  wsc.Queues,
		Timeout: time.Duration(wsc.Timeout) * time.Millisecond,
		ready:   make(chan WaitDone),
	}
}

func (ws WaitSpec) Take(mq *MQ, now time.Time) []*Task {
	tasks := make([]*Task, len(ws.Queues))
	// keep track of last inspected from each queue
	heads := map[*Queue]*Task{}
	// scan queues
	for i, qn := range ws.Queues {
		q := mq.Queues[qn]
		if q == nil {
			tasks[i] = nil
			continue
		}
		t := heads[q]
		if t == nil {
			t = q.Head
		} else {
			t = t.next
		}
		t = q.NextUndispatched(t, now)
		heads[q] = t
		tasks[i] = t
	}
	return tasks
}

func (ws WaitSpec) Ready(mq *MQ, now time.Time) ([]*Task, bool) {
	tasks := make([]*Task, len(ws.Queues))
	// keep track of last inspected from each queue
	heads := map[*Queue]*Task{}
	// scan queues
	for i, qn := range ws.Queues {
		q := mq.Queues[qn]
		if q == nil {
			return nil, false
		}
		t := heads[q]
		if t == nil {
			t = q.Head
		} else {
			t = t.next
		}
		t = q.NextUndispatched(t, now)
		if t == nil {
			return nil, false
		}
		heads[q] = t
		tasks[i] = t
	}
	return tasks, true
}

type Waiters struct {
	Head *WaitSpec
	Tail *WaitSpec
}

func NewWaiters() *Waiters {
	return &Waiters{
		Head: nil,
		Tail: nil,
	}
}

func (wss *Waiters) Remove(ws *WaitSpec) {
	if ws.prev != nil {
		ws.prev.next = ws.next
	}
	if ws.next != nil {
		ws.next.prev = ws.prev
	}
	if wss.Head == ws {
		wss.Head = ws.next
	}
	if wss.Tail == ws {
		wss.Tail = ws.prev
	}
	ws.next = nil
	ws.prev = nil
}

func (wss *Waiters) Append(ws *WaitSpec) {
	if wss.Head == nil {
		wss.Head = ws
		wss.Tail = ws
	} else {
		wss.Tail.next = ws
		ws.prev = wss.Tail
		wss.Tail = ws
	}
}
