package main

import "log"
import "time"
import "container/heap"
import "github.com/satori/go.uuid"

type TaskHeap struct {
	T []*Task
	F func(*Task) time.Time
}

func (t TaskHeap) Len() int           { return len(t.T) }
func (t TaskHeap) Less(i, j int) bool { return t.F(t.T[i]).Before(t.F(t.T[j])) }
func (t TaskHeap) Swap(i, j int)      { t.T[i], t.T[j] = t.T[j], t.T[i] }

func (t *TaskHeap) Push(x interface{}) {
	t.T = append(t.T, x.(*Task))
}

func (t *TaskHeap) Pop() interface{} {
	old := t.T
	n := len(old)
	x := old[n-1]
	t.T = old[0 : n-1]
	return x
}

type MQ struct {
	Tasks    map[uuid.UUID]*Task
	Queues   map[string]*Queue
	ByExpiry *TaskHeap
	ByRetry  *TaskHeap
}

func NewMQ() *MQ {
	return &MQ{
		Tasks:    map[uuid.UUID]*Task{},
		Queues:   map[string]*Queue{},
		ByExpiry: &TaskHeap{F: func(t *Task) time.Time { return t.expiry }},
		ByRetry:  &TaskHeap{F: func(t *Task) time.Time { return t.retryTime }},
	}
}

func (mq *MQ) Add(qn string, t *Task) {
	q := mq.Queues[qn]
	if q == nil {
		q = &Queue{Name: qn}
		mq.Queues[qn] = q
	}
	q.Append(t)
	mq.Tasks[t.ID] = t
	heap.Push(mq.ByExpiry, t)
}

func (mq *MQ) Find(id uuid.UUID, now time.Time) *Task {
	mq.GC(now)
	return mq.Tasks[id]
}

func (mq *MQ) Dispatch(t *Task, now time.Time) {
	mq.GC(now)
	t.Dispatch(now)
	heap.Push(mq.ByRetry, t)
	q := t.q
	q.Remove(t)
	q.Append(t)
}

func (mq *MQ) deleteTask(t *Task) {
	delete(mq.Tasks, t.ID)
	q := t.q
	q.Remove(t)
	if q.Head == nil {
		delete(mq.Queues, q.Name)
	}
}

func (mq *MQ) deleteFromRetry(t *Task) {
	for i, task := range mq.ByRetry.T {
		if task == t {
			heap.Remove(mq.ByRetry, i)
			break
		}
	}
}

func (mq *MQ) deleteFromExpiry(t *Task) {
	for i, task := range mq.ByExpiry.T {
		if task == t {
			heap.Remove(mq.ByExpiry, i)
			break
		}
	}
}

func (mq *MQ) GC(now time.Time) {
	for mq.ByExpiry.Len() > 0 {
		task := mq.ByExpiry.T[0]
		if !task.Expired(now) {
			break
		}
		log.Println("Deleting Task:", task.ID)
		// delete task
		heap.Pop(mq.ByExpiry)
		mq.deleteTask(task)
		mq.deleteFromRetry(task)
	}
	for mq.ByRetry.Len() > 0 {
		task := mq.ByRetry.T[0]
		if !task.NeedRetry(now) {
			break
		}
		log.Println("Task Failed:", task.ID)
		heap.Pop(mq.ByRetry)
		task.Undispatch()
	}
}

func (mq *MQ) DeleteTask(t *Task) {
	mq.deleteTask(t)
	mq.deleteFromRetry(t)
	mq.deleteFromExpiry(t)
}

func (mq *MQ) Failed(t *Task) {
	mq.deleteFromRetry(t)
	t.Undispatch()
}