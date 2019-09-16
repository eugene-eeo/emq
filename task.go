package main

import (
	"github.com/satori/go.uuid"
	"time"
)

type TaskUid uint64

type TaskStatus int

const (
	StatusOk = TaskStatus(iota)
	StatusFail
	StatusTimeout
	StatusExpired
)

type Task struct {
	Id          string        `json:"id"`
	Uid         TaskUid       `json:"-"` // Internal ID
	QueueName   string        `json:"-"`
	Content     interface{}   `json:"content"`
	Retries     int           `json:"retries"`
	JobDuration time.Duration `json:"-"`
	Expiry      time.Duration `json:"-"`
}

type TaskConfig struct {
	Id          string      `json:"id"`
	Content     interface{} `json:"content"`
	Retries     int         `json:"retries"`
	JobDuration int         `json:"job_duration"` // Job duration in seconds
	Expiry      int         `json:"expiry"`       // Job duration in seconds
}

func (tc *TaskConfig) Fill() error {
	if tc.Id == "" {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		tc.Id = id.String()
	}
	return nil
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
	uid    TaskUid
	status TaskStatus
}
