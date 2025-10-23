package services

import (
	"log/slog"
	"scheduler/internal/repositories"
)

type JobService struct {
	log  *slog.Logger
	repo *repositories.JobRepositoryInterface
}

func NewJobService(logger *slog.Logger, repo *repositories.JobRepositoryInterface) *JobService {
	return &JobService{
		log:  logger,
		repo: repo,
	}
}
