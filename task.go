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

type Task struct {
	Id          uuid.UUID     `json:"id"`
	QueueName   string        `json:"-"`
	Content     interface{}   `json:"content"`
	Retries     int           `json:"retries"`
	JobDuration time.Duration `json:"-"`
	Expiry      time.Duration `json:"-"`
}

type TaskConfig struct {
	Id          uuid.UUID   `json:"-"`
	Content     interface{} `json:"content"`
	Retries     int         `json:"retries"`
	JobDuration int         `json:"job_duration"` // Job duration in seconds
	Expiry      int         `json:"expiry"`       // Job duration in seconds
}

func NewTaskFromConfig(tc *TaskConfig, queueName string) *Task {
	return &Task{
		Id:          tc.Id,
		QueueName:   queueName,
		Content:     tc.Content,
		Retries:     tc.Retries,
		JobDuration: time.Duration(tc.JobDuration) * time.Second,
		Expiry:      time.Duration(tc.Expiry) * time.Second,
	}
}

type TaskInfo struct {
	id     uuid.UUID
	status TaskStatus
}
