package handlers

import (
	"scheduler/internal/services"

	"go.uber.org/zap"
)

type Handlers struct {
	log         *zap.Logger
	JobsHandler JobsHandlerInterface
}

func NewHandlers(logger *zap.Logger, service *services.Services) *Handlers {
	return &Handlers{
		log:         logger,
		JobsHandler: NewJobsHandler(logger, service.JobsService),
	}
}
