package app

import (
	"context"
	"scheduler/internal/config"
	httpserver "scheduler/internal/http-server"
	"scheduler/internal/http-server/handlers"
	"scheduler/internal/repositories/postgres"
	"scheduler/internal/services"

	"go.uber.org/zap"
)

func Run(ctx context.Context, log *zap.Logger, config config.ServiceConfig) error {
	storage, err := postgres.NewDatabase(ctx, config.DbConfig.DBConn)
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err))
		return err
	}
	log.Info(config.DbConfig.DBConn)

	dbpool := storage.GetPool()
	defer dbpool.Close()

	jobs_repo := postgres.NewJobsRepository(dbpool)

	services := services.NewServices(log, jobs_repo)

	handlers := handlers.NewHandlers(log, services)

	server := httpserver.NewServer(log, handlers, config)
	if err := server.Run(); err != nil {
		log.Error("Failed to run HTTP server", zap.Error(err))
		return err
	}
	return nil
}
