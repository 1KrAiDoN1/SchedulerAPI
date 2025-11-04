package inmemory

import (
	"context"
	"scheduler/internal/domain/entity"
	"scheduler/internal/repositories"
	"sync"
)

type JobsRepository struct {
	jobs map[string]*entity.Job
	mu   sync.RWMutex
}

func NewJobsRepository() *JobsRepository {
	return &JobsRepository{}
}

func (j *JobsRepository) Create(ctx context.Context, job *entity.Job) error {
	j.mu.RLock()
	defer j.mu.RUnlock()

	j.jobs[job.ID] = job

	return nil
}
func (j *JobsRepository) Read(ctx context.Context, jobID string) (*entity.Job, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	job, ok := j.jobs[jobID]
	if !ok {
		return nil, repositories.ErrJobNotFound
	}

	return job, nil
}
