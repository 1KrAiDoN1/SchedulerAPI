package handlers

import (
	"log/slog"
	"scheduler/internal/services"
)

type Handlers struct {
	log *slog.Logger
	JobsHandlerInterface
}

func NewHandlers(logger *slog.Logger, service *services.Services) *Handlers {
	return &Handlers{
		log:                  logger,
		JobsHandlerInterface: NewJobHandler(logger, service.JobServiceInterface),
	}
}
