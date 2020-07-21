package main

import "time"
import "github.com/satori/go.uuid"

// TaskConfig is the schema for the JSON blob used to create tasks.
type TaskConfig struct {
	Retry   uint64      `json:"retry"`
	Expiry  uint64      `json:"expiry"`
	Content interface{} `json:"content"`
}

func (tc TaskConfig) ToTask(id uuid.UUID) Task {
	return Task{
		ID:      id,
		Content: tc.Content,
		retry:   time.Second * time.Duration(tc.Retry),
		expiry:  time.Now().Add(time.Second * time.Duration(tc.Expiry)),
	}
}

type Task struct {
	ID      uuid.UUID   `json:"id"`
	Content interface{} `json:"content"`
	// timing
	expiry     time.Time
	retry      time.Duration
	retryTime  time.Time
	dispatched bool
	// linked list
	next *Task
	prev *Task
	q    *Queue
}

// InDispatch returns whether a task is in dispatch,
// i.e. it has been dispatched and hasn't exceeded the
// Expiry TTL or Retry TTL yet
func (t *Task) InDispatch(now time.Time) bool {
	return !t.Expired(now) && t.dispatched && now.Before(t.retryTime)
}

// CanDispatch returns whether a task can be dispatched
func (t *Task) CanDispatch(now time.Time) bool {
	return !t.Expired(now) && (!t.dispatched || t.retryTime.Before(now))
}

// Dispatch marks the task as dispatched
func (t *Task) Dispatch(now time.Time) {
	t.dispatched = true
	t.retryTime = now.Add(t.retry)
}

// Undispatch marks the task as undispatched
func (t *Task) Undispatch() {
	t.dispatched = false
}

// Expired checks if the task has expired
func (t *Task) Expired(now time.Time) bool {
	return now.After(t.expiry)
}

// NeedRetry checks if the task needs retry
func (t *Task) NeedRetry(now time.Time) bool {
	return t.dispatched && now.After(t.retryTime) && !t.Expired(now)
}

type Queue struct {
	Name string
	Head *Task
	Tail *Task
}

func (q *Queue) Append(t *Task) {
	t.q = q
	if q.Head == nil {
		q.Head = t
		q.Tail = t
	} else {
		q.Tail.next = t
		t.prev = q.Tail
		q.Tail = t
	}
}

func (q *Queue) Remove(t *Task) {
	if t.prev != nil {
		t.prev.next = t.next
	}
	if t.next != nil {
		t.next.prev = t.prev
	}
	if q.Head == t {
		q.Head = t.next
	}
	if q.Tail == t {
		q.Tail = t.prev
	}
	t.next = nil
	t.prev = nil
	t.q = nil
}

func (q *Queue) NextUndispatched(t *Task, now time.Time) *Task {
	for ; t != nil; t = t.next {
		if t.CanDispatch(now) {
			return t
		}
	}
	return nil
}
