package services

import (
	"scheduler/internal/repositories"

	"go.uber.org/zap"
)

type Services struct {
	log         *zap.Logger
	JobsService JobServiceInterface
}

func NewServices(logger *zap.Logger, jobs_repo repositories.JobRepositoryInterface) *Services {
	return &Services{
		log:         logger,
		JobsService: NewJobsService(logger, jobs_repo),
	}

}
