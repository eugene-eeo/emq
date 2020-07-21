package main

import "time"
import "github.com/satori/go.uuid"

// TaskConfig is the schema for the JSON blob used to create tasks.
type TaskConfig struct {
	Retry   *uint32     `json:"retry"`
	Expiry  *uint32     `json:"expiry"`
	Content interface{} `json:"content"`
}

func (tc TaskConfig) ToTask(id uuid.UUID, now time.Time) Task {
	t := Task{
		ID:      id,
		Content: tc.Content,
		retry:   time.Minute * 5,
		expiry:  now.Add(time.Hour * 24),
	}
	if tc.Retry != nil {
		t.retry = time.Second * time.Duration(*tc.Retry)
	}
	if tc.Expiry != nil {
		t.expiry = now.Add(time.Second * time.Duration(*tc.Expiry))
	}
	return t
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

// Next returns the next task in the linked list, if any
func (t *Task) Next() *Task {
	if t.next == &t.q.root {
		return nil
	}
	return t.next
}

type Queue struct {
	Name string
	root Task
}

func (q *Queue) Init() *Queue {
	q.root.next = &q.root
	q.root.prev = &q.root
	return q
}

func (q *Queue) Append(t *Task) {
	t.q = q
	at := q.root.prev // append to this node
	t.next = at.next
	t.prev = at
	at.next.prev = t
	at.next = t
}

func (q *Queue) Remove(t *Task) {
	t.prev.next = t.next
	t.next.prev = t.prev
	t.next = nil
	t.prev = nil
	t.q = nil
}

func (q *Queue) Empty() bool {
	return q.root.next == &q.root
}

func (q *Queue) Head() *Task {
	if q.Empty() {
		return nil
	}
	return q.root.next
}

func (q *Queue) NextUndispatched(t *Task, now time.Time) (curr *Task, next *Task) {
	for ; t != nil; t = t.Next() {
		if t.CanDispatch(now) {
			return t, t.Next()
		}
	}
	return nil, nil
}
