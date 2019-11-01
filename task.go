package main

import (
	"github.com/satori/go.uuid"
	"time"
)

type TaskStatus int

const (
	StatusDone = TaskStatus(iota)
	StatusFail
	StatusTimeout
	StatusExpired
)

type TaskId = uuid.UUID

type Task struct {
	Id          TaskId        `json:"id"`
	Queue       *Queue        `json:"-"`
	Content     interface{}   `json:"content"`
	Retries     int           `json:"retries"`
	Received    time.Time     `json:"received"`
	JobDuration time.Duration `json:"-"`
	Expiry      time.Duration `json:"-"`
	prev        *Task         `json:"-"`
	next        *Task         `json:"-"`
	dispatched  bool
}

type TaskConfig struct {
	Id          TaskId      `json:"-"`
	Content     interface{} `json:"content"`
	Retries     int         `json:"retries"`
	JobDuration int         `json:"job_duration"`
	Expiry      int         `json:"expiry"`
}

func NewTaskFromConfig(tc *TaskConfig, queue *Queue) *Task {
	return &Task{
		Id:          tc.Id,
		Queue:       queue,
		Content:     tc.Content,
		Retries:     tc.Retries,
		Received:    time.Now().UTC(),
		JobDuration: time.Duration(tc.JobDuration) * time.Second,
		Expiry:      time.Duration(tc.Expiry) * time.Second,
	}
}

type TaskInfo struct {
	id     TaskId
	status TaskStatus
}
