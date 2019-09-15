package main

import "log"

const MAX_QUEUE_SIZE int = 1024

type Queue struct {
	Name  string
	tasks []*Task
	size  int
	head  int
	tail  int
	count int
}

func NewQueue(name string) *Queue {
	return &Queue{
		Name:  name,
		tasks: make([]*Task, 2),
		size:  2,
		head:  0,
		tail:  0,
		count: 0,
	}
}

func (q *Queue) Print() {
	log.Print("================")
	log.Printf("Head: %d", q.head)
	log.Printf("Tail: %d", q.tail)
	log.Printf("Count: %d", q.count)
	x := []string{}
	for _, task := range q.tasks {
		if task == nil {
			x = append(x, "<nil>")
		} else {
			x = append(x, task.Id)
		}
	}
	log.Print(x)
}

func (q *Queue) full() bool {
	return q.size-1 == q.count
}

func (q *Queue) Enqueue(t *Task) bool {
	//q.Print()
	if q.full() {
		if q.size >= MAX_QUEUE_SIZE {
			return false
		}
		old_size := q.size
		new_size := (q.size-1)*2 + 1
		tasks := make([]*Task, new_size)
		// Case 1: simple extension will do
		if q.head <= q.tail {
			copy(tasks, q.tasks)
		} else {
			// Case 2:
			// Before: | .... | tail | null | head | ...  |
			// After:  | .... | tail | null | .... | null | head | .... |
			new_head := new_size - (old_size - q.head)
			copy(tasks, q.tasks[:q.tail+1])
			copy(tasks[new_head:], q.tasks[q.head:])
			q.head = new_head
		}
		q.tasks = tasks
		q.size = new_size
	}
	q.count++
	q.tasks[q.tail] = t
	q.tail = (q.tail + 1) % q.size
	//q.Print()
	return true
}

func (q *Queue) Dequeue() *Task {
	//q.Print()
	task := q.tasks[q.head]
	if task == nil {
		return nil
	}
	q.tasks[q.head] = nil
	q.count--
	q.head = (q.head + 1) % len(q.tasks)
	//q.Print()
	return task
}
