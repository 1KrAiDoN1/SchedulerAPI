package services

import (
	"context"
	"scheduler/internal/domain/entity"
	"scheduler/internal/repositories"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type JobsService struct {
	log       *zap.Logger
	jobs_repo repositories.JobRepositoryInterface
}

func NewJobsService(logger *zap.Logger, jobs_repo repositories.JobRepositoryInterface) *JobsService {
	return &JobsService{
		log:       logger,
		jobs_repo: jobs_repo,
	}
}

func (j *JobsService) Start() {

}

func (j *JobsService) CreateJob(ctx context.Context, job *entity.Job) (string, error) {
	job.ID = uuid.NewString()
	return "", nil
}
