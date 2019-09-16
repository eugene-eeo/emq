package main

type Waiters struct {
	head *Waiter
	tail *Waiter
}

func (ws *Waiters) AddWaiter(w *Waiter, queues map[string]*Queue) {
	for _, name := range w.Queues {
		q := queues[name]
		if q != nil {
			w.Update(q.Dequeue())
		}
	}
	// Only add to the list of waiters if we cannot satisfy it immediately
	if w.IsReady() {
		w.EmitReady()
	} else {
		if ws.tail == nil {
			ws.head = w
			ws.tail = w
		} else {
			ws.tail.Next = w
			ws.tail = w
		}
	}
}

func (ws *Waiters) Update(q *Queue, t *Task) {
	// Consume a task if possible
	var prev *Waiter = nil
	curr := ws.head
	for curr != nil {
		if curr.Update(t) {
			if curr.IsReady() {
				if prev != nil {
					prev.Next = curr.Next
				} else {
					ws.head = curr.Next
				}
				if curr == ws.tail {
					ws.tail = nil
				}
				curr.EmitReady()
			}
			return
		}
		// Advance
		prev = curr
		curr = curr.Next
	}
	// Otherwise enqueue it
	q.Enqueue(t)
}

type WaitConfigJson struct {
	NoWait bool     `json:"no_wait"`
	Queues []string `json:"queues"`
}

type Waiter struct {
	NoWait bool
	Queues []string
	Tasks  []*Task
	Ready  chan bool
	Next   *Waiter
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

func (w *Waiter) IsReady() bool {
	if w.NoWait {
		return true
	}
	for _, x := range w.Tasks {
		if x == nil {
			return false
		}
	}
	return true
}

func (w *Waiter) Update(t *Task) bool {
	if t == nil {
		return false
	}
	for i, name := range w.Queues {
		if name == t.QueueName && w.Tasks[i] == nil {
			w.Tasks[i] = t
			return true
		}
	}
	return false
}
