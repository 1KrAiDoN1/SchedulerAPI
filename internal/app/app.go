package app

import (
	"context"
	"log/slog"
	"scheduler/internal/config"
	"scheduler/internal/handlers"
	httpserver "scheduler/internal/http-server"
	"scheduler/internal/repositories/postgres"
	"scheduler/internal/services"
	"scheduler/pkg/lib/slogger"
)

func Run(ctx context.Context, log *slog.Logger, config config.ServiceConfig) error {
	storage, err := postgres.NewDatabase(ctx, config.DbConfig.DBConn)
	if err != nil {
		log.Error("Failed to connect to database", slogger.Err(err))
		return err
	}
	log.Info(config.DbConfig.DBConn)

	dbpool := storage.GetPool()
	defer dbpool.Close()

	services := services.NewServices(log, storage)

	handlers := handlers.NewHandlers(log, services)

	server := httpserver.NewServer(log, handlers, config)
	if err := server.Run(); err != nil {
		log.Error("Failed to run HTTP server", slogger.Err(err))
		return err
	}
	return nil
}
