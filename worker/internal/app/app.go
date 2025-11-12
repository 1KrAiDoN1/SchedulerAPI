package app

import (
	"context"
	"fmt"
	"scheduler/internal/models/dto"
	"scheduler/worker/internal/config"
	"scheduler/worker/internal/services"
	"scheduler/worker/internal/services/publisher"
	"scheduler/worker/internal/services/subscriber"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func Run(ctx context.Context, log *zap.Logger, cfg config.Config) error {
	// Генерируем уникальный ID для воркера
	workerID := uuid.NewString()
	log.Info("Generated worker ID", zap.String("worker_id", workerID))

	// Создаем publisher для отправки результатов
	executionPublisher, err := publisher.NewNATSExecutionPublisher(ctx, log, cfg.NATSURL)
	if err != nil {
		log.Error("Failed to create execution publisher", zap.Error(err))
		return fmt.Errorf("create execution publisher: %w", err)
	}
	defer executionPublisher.Close()

	// Создаем subscriber для получения джоб
	jobSubscriber, err := subscriber.NewNATSJobSubscriber(ctx, log, cfg.NATSURL, workerID)
	if err != nil {
		log.Error("Failed to create job subscriber", zap.Error(err))
		return fmt.Errorf("create job subscriber: %w", err)
	}
	defer jobSubscriber.Close()

	// Создаем job processor для обработки джоб
	jobProcessor := services.NewJobProcessor(executionPublisher, log, workerID)

	log.Info("Worker started",
		zap.String("worker_id", workerID),
		zap.String("nats_url", cfg.NATSURL))

	// Запускаем subscriber для получения джоб
	// Subscribe блокирует выполнение до отмены контекста
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
		log.Error("Failed to subscribe to jobs", zap.Error(err))
		return fmt.Errorf("subscribe to jobs: %w", err)
	}

	log.Info("Worker stopped", zap.String("worker_id", workerID))
	return nil
}
