package main

import "time"
import "log"

type TaskInfo struct {
	task   *Task
	status TaskStatus
}

type Dispatched struct {
	statuses map[*Task](chan TaskStatus)
	fwd      chan TaskInfo
}

func NewDispatched(fwd chan TaskInfo) *Dispatched {
	return &Dispatched{
		statuses: map[*Task](chan TaskStatus){},
		fwd:      fwd,
	}
}

func (d *Dispatched) Untrack(t *Task) {
	delete(d.statuses, t)
}

func (d *Dispatched) Track(t *Task) {
	c := make(chan TaskStatus, 1)
	d.statuses[t] = c
	go func() {
		timer := time.NewTimer(time.Second * time.Duration(t.JobDuration))
		for {
			select {
			case <-timer.C:
				timer.Stop()
				// Check if we really expired
				if t.JobDuration > 0 {
					log.Print("Timeout")
					d.fwd <- TaskInfo{t, StatusFail}
					return
				}
			case status := <-c:
				timer.Stop()
				d.fwd <- TaskInfo{t, status}
				return
			}
		}
	}()
}

func (d *Dispatched) Put(t *Task, status TaskStatus) {
	ch := d.statuses[t]
	if ch != nil {
		ch <- status
	}
}
