package main

type Waiters struct {
	head *Waiter
	tail *Waiter
}

func (ws *Waiters) AddWaiter(w *Waiter) {
	if ws.tail == nil {
		ws.head = w
		ws.tail = w
	} else {
		ws.tail.Next = w
		ws.tail = w
	}
}

func (ws *Waiters) Update(queues map[string]*Queue) {
	var prev *Waiter = nil
	curr := ws.head
	for curr != nil {
		if curr.IsReady(queues) {
			if prev != nil {
				prev.Next = curr.Next
			} else {
				ws.head = curr.Next
			}
			if curr == ws.tail {
				ws.tail = nil
			}
			curr.Consume(queues)
			curr.EmitReady()
		}
		// Advance
		prev = curr
		curr = curr.Next
	}
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
