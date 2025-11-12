package handlers

import (
	handlers "scheduler/internal/domain/handler"
	"scheduler/internal/services"

	"go.uber.org/zap"
)

type Handlers struct {
	log         *zap.Logger
	JobsHandler handlers.JobsHandlerInterface
}

func NewHandlers(logger *zap.Logger, service *services.JobsService) *Handlers {
	return &Handlers{
		log:         logger,
		JobsHandler: NewJobsHandler(logger, service),
	}
}
