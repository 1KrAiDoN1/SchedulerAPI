package app

import (
	"context"
	"fmt"
	"os"
	"scheduler/internal/config"
	"scheduler/internal/domain/entity"
	httpserver "scheduler/internal/http-server"
	"scheduler/internal/http-server/handlers"
	"scheduler/internal/repositories/postgres"
	"scheduler/internal/repositories/publisher"
	"scheduler/internal/repositories/subscriber"
	"scheduler/internal/services"
	"time"

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

	rawInterval := os.Getenv("SCHEDULER_INTERVAL")
	interval, err := time.ParseDuration(rawInterval)
	if err != nil {
		return fmt.Errorf("parse interval: %w", err)
	}

	natsPublisher, err := publisher.NewNATSJobPublisher(ctx, log, config.NATSConfig.URL)
	if err != nil {
		log.Error("Failed to connect to NATS publisher", zap.Error(err))
		return err
	}
	defer natsPublisher.Close()

	natsSubscriber, err := subscriber.NewNATSJobSubscriber(ctx, log, config.NATSConfig.URL)
	if err != nil {
		log.Error("Failed to connect to NATS subscriber", zap.Error(err))
		return err
	}
	defer natsSubscriber.Close()

	jobs_repo := postgres.NewJobsRepository(dbpool)

	executions_repo := postgres.NewExecutionsRepository(dbpool)

	service := services.NewJobsService(jobs_repo, executions_repo, natsPublisher, interval, log)

	// Запускаем subscriber для обработки результатов от воркеров
	subscriberCtx, subscriberCancel := context.WithCancel(ctx)
	defer subscriberCancel()

	go func() {
		if err := natsSubscriber.Subscribe(subscriberCtx, func(ctx context.Context, execution *entity.Execution) error {
			return service.HandleExecutionResult(ctx, execution)
		}); err != nil {
			log.Error("Failed to subscribe to execution results", zap.Error(err))
		}
	}()

	go func() {
		if err := service.Start(context.Background()); err != nil {
			log.Error("Scheduler tick loop failed", zap.Error(err))
		}
	}()

	handlers := handlers.NewHandlers(log, service)

	server := httpserver.NewServer(log, handlers, config)
	if err := server.Run(); err != nil {
		log.Error("Failed to run HTTP server", zap.Error(err))
		return err
	}
	return nil
}
