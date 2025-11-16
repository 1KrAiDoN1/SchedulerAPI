package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"scheduler/internal/config"
	"scheduler/internal/domain/entity"
	httpserver "scheduler/internal/http-server"
	"scheduler/internal/http-server/handlers"
	"scheduler/internal/repositories/postgres"
	"scheduler/internal/repositories/publisher"
	"scheduler/internal/repositories/subscriber"
	"scheduler/internal/services"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func Run(ctx context.Context, log *zap.Logger, config config.ServiceConfig) error {
	// Создаем контекст с возможностью отмены
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Обрабатываем системные сигналы для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Подключаемся к базе данных
	storage, err := postgres.NewDatabase(ctx, config.DbConfig.DBConn)
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err))
		return fmt.Errorf("database connection failed: %w", err)
	}
	log.Info("Connected to database", zap.String("dsn", config.DbConfig.DBConn))

	dbpool := storage.GetPool()
	defer func() {
		log.Info("Closing database connection...")
		dbpool.Close()
		log.Info("Database connection closed")
	}()

	// Парсим интервал планировщика
	rawInterval := os.Getenv("SCHEDULER_INTERVAL")
	interval, err := time.ParseDuration(rawInterval)
	if err != nil {
		return fmt.Errorf("parse scheduler interval: %w", err)
	}
	log.Info("Scheduler interval configured", zap.Duration("interval", interval))

	// Подключаемся к NATS publisher
	natsPublisher, err := publisher.NewNATSJobPublisher(ctx, log, config.NATSConfig.URL)
	if err != nil {
		log.Error("Failed to connect to NATS publisher", zap.Error(err))
		return fmt.Errorf("nats publisher connection failed: %w", err)
	}
	defer func() {
		log.Info("Closing NATS publisher...")
		err := natsPublisher.Close()
		if err != nil {
			log.Error("Failed to close NATS publisher", zap.Error(err))
		}
		log.Info("NATS publisher closed")
	}()

	// Подключаемся к NATS subscriber
	natsSubscriber, err := subscriber.NewNATSJobSubscriber(ctx, log, config.NATSConfig.URL)
	if err != nil {
		log.Error("Failed to connect to NATS subscriber", zap.Error(err))
		return fmt.Errorf("nats subscriber connection failed: %w", err)
	}
	defer func() {
		log.Info("Closing NATS subscriber...")
		err := natsSubscriber.Close()
		if err != nil {
			log.Error("Failed to close NATS subscriber", zap.Error(err))
		}
		log.Info("NATS subscriber closed")
	}()

	// Создаем репозитории
	jobs_repo := postgres.NewJobsRepository(dbpool)
	executions_repo := postgres.NewExecutionsRepository(dbpool)

	// Создаем сервис
	service := services.NewJobsService(jobs_repo, executions_repo, natsPublisher, interval, log)

	// Запускаем subscriber для обработки результатов от воркеров
	subscriberDone := make(chan struct{})
	go func() {
		defer close(subscriberDone)
		log.Info("Starting execution results subscriber...")
		if err := natsSubscriber.Subscribe(ctx, func(ctx context.Context, execution *entity.Execution) error {
			return service.HandleExecutionResult(ctx, execution)
		}); err != nil {
			if ctx.Err() == nil { // Игнорируем ошибки при отмене контекста
				log.Error("Subscriber stopped with error", zap.Error(err))
			}
		}
		log.Info("Execution results subscriber stopped")
	}()

	// Запускаем планировщик задач
	schedulerDone := make(chan struct{})
	go func() {
		defer close(schedulerDone)
		log.Info("Starting job scheduler...")
		if err := service.Start(ctx); err != nil {
			if ctx.Err() == nil { // Игнорируем ошибки при отмене контекста
				log.Error("Scheduler stopped with error", zap.Error(err))
			}
		}
		log.Info("Job scheduler stopped")
	}()

	// Создаем HTTP handlers и сервер
	jobsHandler := handlers.NewJobsHandler(log, service)

	server := httpserver.NewServer(log, jobsHandler, config)

	// Запускаем HTTP сервер в отдельной горутине
	serverDone := make(chan error, 1)
	go func() {
		log.Info("Starting HTTP server...")
		if err := server.Run(); err != nil {
			serverDone <- err
		}
		close(serverDone)
	}()

	// Ожидаем сигнал завершения или ошибку сервера
	select {
	case sig := <-sigChan:
		log.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel() // Отменяем контекст для остановки всех горутин

		// Останавливаем HTTP сервер
		if err := server.Shutdown(); err != nil {
			log.Error("Failed to shutdown HTTP server", zap.Error(err))
		}

		// Ждем завершения всех горутин
		log.Info("Waiting for goroutines to finish...")
		<-subscriberDone
		<-schedulerDone

		log.Info("Application gracefully shut down")
		return nil

	case err := <-serverDone:
		if err != nil {
			log.Error("HTTP server stopped with error", zap.Error(err))
			cancel()
			return err
		}
		return nil
	}
}
