package repositories

import (
	"context"
	"errors"
	"scheduler/internal/domain/entity"
)

var (
	ErrJobNotFound = errors.New("job not found")
)

type JobRepositoryInterface interface {
	Create(ctx context.Context, job *entity.Job) error
	Read(ctx context.Context, jobID string) (*entity.Job, error)
}
