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

type TaskUid = uuid.UUID

type Task struct {
	Id          TaskUid       `json:"id"`
	Queue       *Queue        `json:"-"`
	Content     interface{}   `json:"content"`
	Retries     int           `json:"retries"`
	JobDuration time.Duration `json:"-"`
	Expiry      time.Duration `json:"-"`
	prev        *Task         `json:"-"`
	next        *Task         `json:"-"`
}

type TaskConfig struct {
	Id          TaskUid     `json:"-"`
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
		JobDuration: time.Duration(tc.JobDuration) * time.Second,
		Expiry:      time.Duration(tc.Expiry) * time.Second,
	}
}

type TaskInfo struct {
	id     TaskUid
	status TaskStatus
}
