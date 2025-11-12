package entity

import (
	"context"
	"time"
)

type JobStatus string

const (
	StatusQueued    JobStatus = "queued"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

// Job представляет структуру задачи
type Job struct {
	ID             string
	Kind           JobKind
	Status         JobStatus
	Interval       *time.Duration
	Once           *int64
	LastFinishedAt int64
	Payload        any
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
