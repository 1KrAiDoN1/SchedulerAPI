package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"scheduler/internal/models/dto"
	"scheduler/worker/internal/config"
	"scheduler/worker/internal/services"
	"scheduler/worker/internal/services/publisher"
	"scheduler/worker/internal/services/subscriber"
	"syscall"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func Run(ctx context.Context, log *zap.Logger, cfg config.Config) error {
	// Создаем контекст с возможностью отмены
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Обрабатываем системные сигналы для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Генерируем уникальный ID для воркера
	workerID := uuid.NewString()
	log.Info("Generated worker ID", zap.String("worker_id", workerID))

	// Создаем publisher для отправки результатов
	executionPublisher, err := publisher.NewNATSExecutionPublisher(ctx, log, cfg.NATSURL)
	if err != nil {
		log.Error("Failed to create execution publisher", zap.Error(err))
		return fmt.Errorf("create execution publisher: %w", err)
	}
	defer func() {
		log.Info("Closing execution publisher...")
		executionPublisher.Close()
		log.Info("Execution publisher closed")
	}()

	// Создаем subscriber для получения джоб
	jobSubscriber, err := subscriber.NewNATSJobSubscriber(ctx, log, cfg.NATSURL, workerID)
	if err != nil {
		log.Error("Failed to create job subscriber", zap.Error(err))
		return fmt.Errorf("create job subscriber: %w", err)
	}
	defer func() {
		log.Info("Closing job subscriber...")
		jobSubscriber.Close()
		log.Info("Job subscriber closed")
	}()

	// Создаем job processor для обработки джоб
	jobProcessor := services.NewJobProcessor(executionPublisher, log, workerID)

	log.Info("Worker started",
		zap.String("worker_id", workerID),
		zap.String("nats_url", cfg.NATSURL))

	// Запускаем subscriber для получения джоб в отдельной горутине
	subscriberDone := make(chan error, 1)
	go func() {
		if err := jobSubscriber.Subscribe(ctx, func(ctx context.Context, job *dto.JobDTO) error {
			// Обрабатываем джобу в отдельной горутине для параллельной обработки
			go func() {
				if err := jobProcessor.ProcessJob(ctx, job); err != nil {
					log.Error("Failed to process job",
						zap.String("job_id", job.ID),
						zap.Error(err))
				}
			}()
			return nil
		}); err != nil {
			if ctx.Err() == nil { // Игнорируем ошибки при отмене контекста
				subscriberDone <- fmt.Errorf("subscribe to jobs: %w", err)
			}
		}
		close(subscriberDone)
	}()

	// Ожидаем сигнал завершения или ошибку subscriber
	select {
	case sig := <-sigChan:
		log.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel() // Отменяем контекст для остановки всех горутин

		// Ждем завершения subscriber
		log.Info("Waiting for subscriber to finish...")
		<-subscriberDone

		log.Info("Worker gracefully shut down", zap.String("worker_id", workerID))
		return nil

	case err := <-subscriberDone:
		if err != nil {
			log.Error("Subscriber stopped with error", zap.Error(err))
			cancel()
			return err
		}
		log.Info("Worker stopped", zap.String("worker_id", workerID))
		return nil
	}
}
