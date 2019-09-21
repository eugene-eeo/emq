package main

const MAX_QUEUE_SIZE int = 1025

type Queue struct {
	Name  string
	head  *Task
	tail  *Task
	count int
}

func NewQueue(name string) *Queue {
	return &Queue{Name: name}
}

func (q *Queue) Enqueue(t *Task) {
	q.count++
	t.prev = q.tail
	if q.tail == nil {
		q.head = t
		q.tail = t
	} else {
		q.tail.next = t
		q.tail = t
	}
}

func (q *Queue) Remove(t *Task) {
	prev := t.prev
	next := t.next
	if prev != nil {
		prev.next = next
	} else {
		q.head = next
	}
	if next != nil {
		next.prev = prev
	} else {
		q.tail = prev
	}
	// Clear for GC
	t.next = nil
	t.prev = nil
}

func (q *Queue) Dequeue() *Task {
	head := q.head
	if head == nil {
		return nil
	}
	q.Remove(head)
	return head
}
