package services

import (
	"context"
	"scheduler/internal/domain/entity"
)

type JobServiceInterface interface {
	CreateJob(ctx context.Context, job *entity.Job) (string, error)
	GetJob(ctx context.Context, jobID string) (*entity.Job, error)
	GetJobs(ctx context.Context) ([]*entity.Job, error)
	DeleteJob(ctx context.Context, jobID string) error
	GetJobExecutions(ctx context.Context, jobID string) ([]*entity.Execution, error)
	CompleteJob(ctx context.Context, jobID string, success bool) error
	HandleExecutionResult(ctx context.Context, execution *entity.Execution) error
	Start(ctx context.Context) error
}
