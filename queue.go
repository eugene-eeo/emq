package main

const MAX_QUEUE_SIZE int = 1025

type QueueNode struct {
	task *Task
	prev *QueueNode
	next *QueueNode
}

type Queue struct {
	Name  string
	head  *QueueNode
	tail  *QueueNode
	count int
}

func NewQueue(name string) *Queue {
	return &Queue{Name: name}
}

func (q *Queue) Enqueue(t *Task) *QueueNode {
	q.count++
	qn := &QueueNode{
		task: t,
		prev: q.tail,
	}
	if q.tail == nil {
		q.head = qn
		q.tail = qn
	} else {
		q.tail.next = qn
		q.tail = qn
	}
	return qn
}

func (q *Queue) Remove(qn *QueueNode) {
	prev := qn.prev
	next := qn.next
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
	qn.next = nil
	qn.prev = nil
}

func (q *Queue) Dequeue() *Task {
	head := q.head
	if head == nil {
		return nil
	}
	q.Remove(head)
	return head.task
}
