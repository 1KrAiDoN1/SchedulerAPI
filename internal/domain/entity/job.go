package entity

import "context"

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Job представляет структуру задачи
type Job struct {
	ID             string      `json:"id"`
	Once           string      `json:"once,omitempty"`
	Interval       string      `json:"interval,omitempty"`
	Status         Status      `json:"status"`
	CreatedAt      int64       `json:"createdAt"`
	LastFinishedAt int64       `json:"lastFinishedAt"`
	Payload        interface{} `json:"payload"`
}

type RunningJob struct {
	*Job

	Cancel context.CancelFunc
}

type JobKind uint8

const (
	JobUndefined = iota
	JobKindInterval
	JobKindOnce
)
