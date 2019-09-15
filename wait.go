package main

type Waiters []*Waiter

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
		*ws = append(*ws, w)
	}
}

func (ws *Waiters) Update(q *Queue, t *Task) {
	// Consume a task if possible
	if len(*ws) > 0 {
		top := (*ws)[0]
		top.Update(t)
		if top.IsReady() {
			top.EmitReady()
			copy((*ws)[0:], (*ws)[1:])
			(*ws)[len(*ws)-1] = nil
			*ws = (*ws)[:len(*ws)-1]
			return
		}
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

func (w *Waiter) Update(t *Task) {
	if t == nil {
		return
	}
	found := false
	for i, name := range w.Queues {
		if !found && name == t.QueueName && w.Tasks[i] == nil {
			w.Tasks[i] = t
			found = true
		}
	}
}
