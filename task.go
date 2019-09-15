package main

import (
	"github.com/satori/go.uuid"
)

type TaskStatus int

const (
	StatusOk = TaskStatus(iota)
	StatusFail
	StatusTimeout
)

type Task struct {
	Id          string      `json:"id"`
	QueueName   string      `json:"-"`
	Content     interface{} `json:"content"`
	Retries     int         `json:"retries"`
	JobDuration int         `json:"job_duration"`
}

type TaskConfig struct {
	Id          string      `json:"id"`
	Content     interface{} `json:"content"`
	Retries     int         `json:"retries"`
	JobDuration int         `json:"job_duration"` // Job duration in seconds
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
		JobDuration: tc.JobDuration,
	}
}
