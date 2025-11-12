package repositories

import (
	"context"
	"errors"
	"scheduler/internal/domain/entity"
)

var (
	ErrJobNotFound       = errors.New("job not found")
	ErrExecutionNotFound = errors.New("execution not found")
)

type JobRepositoryInterface interface {
	Create(ctx context.Context, job *entity.Job) error
	Read(ctx context.Context, jobID string) (*entity.Job, error)
	List(ctx context.Context) ([]*entity.Job, error)
	Upsert(ctx context.Context, jobs []*entity.Job) error
	Delete(ctx context.Context, jobID string) error
}

type JobPublisherInterface interface {
	Publish(ctx context.Context, job *entity.Job) error
}

type JobSubscriberInterface interface {
	Subscribe(ctx context.Context, handler func(ctx context.Context, execution *entity.Execution) error) error
	Close() error
}

type ExecutionRepositoryInterface interface {
	Create(ctx context.Context, execution *entity.Execution) error
	GetByJobID(ctx context.Context, jobID string) ([]*entity.Execution, error)
}
