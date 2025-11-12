package inmemory

import (
	"context"
	"scheduler/internal/domain/entity"
	repositories "scheduler/internal/domain/repository"
	"sync"
)

type JobsRepository struct {
	jobs map[string]*entity.Job
	mu   sync.RWMutex
}

func NewJobsRepository() *JobsRepository {
	return &JobsRepository{
		jobs: make(map[string]*entity.Job),
	}
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

func (j *JobsRepository) List(ctx context.Context) ([]*entity.Job, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	out := make([]*entity.Job, 0, len(j.jobs))

	for _, j := range j.jobs {
		out = append(out, j)
	}
	if len(out) == 0 {
		return nil, repositories.ErrJobNotFound
	}

	return out, nil
}

func (j *JobsRepository) Upsert(ctx context.Context, jobs []*entity.Job) error {
	j.mu.RLock()
	defer j.mu.RUnlock()

	for _, job := range jobs {
		j.jobs[job.ID] = job
	}

	return nil
}
