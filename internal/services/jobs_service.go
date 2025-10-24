package services

import (
	"log/slog"
	"scheduler/internal/repositories"
)

type JobService struct {
	log  *slog.Logger
	repo repositories.Repositories
}

func NewJobService(logger *slog.Logger, repo repositories.Repositories) *JobService {
	return &JobService{
		log:  logger,
		repo: repo,
	}
}
