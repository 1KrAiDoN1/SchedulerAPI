package services

import (
	"log/slog"
	"scheduler/internal/repositories"
)

type Services struct {
	log *slog.Logger
	JobServiceInterface
}

func NewServices(logger *slog.Logger, repo repositories.Repositories) *Services {
	return &Services{
		log:                 logger,
		JobServiceInterface: NewJobService(logger, repo),
	}

}
