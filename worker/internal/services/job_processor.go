package services

import (
	"context"
	"fmt"
	"scheduler/internal/domain/entity"
	"scheduler/internal/models/dto"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type JobProcessor struct {
	publisher ExecutionPublisherInterface
	logger    *zap.Logger
	workerID  string
}

type ExecutionPublisherInterface interface {
	Publish(ctx context.Context, execution *entity.Execution) error
}

func NewJobProcessor(publisher ExecutionPublisherInterface, logger *zap.Logger, workerID string) *JobProcessor {
	return &JobProcessor{
		publisher: publisher,
		logger:    logger,
		workerID:  workerID,
	}
}

// ProcessJob обрабатывает джобу и отправляет результат обратно в планировщик
func (p *JobProcessor) ProcessJob(ctx context.Context, job *dto.JobDTO) error {
	executionID := uuid.NewString()
	startedAt := time.Now().UnixMilli()

	p.logger.Info("Processing job",
		zap.String("job_id", job.ID),
		zap.String("execution_id", executionID),
		zap.String("worker_id", p.workerID),
		zap.Any("payload", job.Payload))

	// Создаем execution со статусом "running"
	execution := &entity.Execution{
		ID:        executionID,
		JobID:     job.ID,
		WorkerID:  p.workerID,
		Status:    "running",
		StartedAt: startedAt,
	}

	// Симуляция обработки джобы
	// В реальном приложении здесь будет логика выполнения джобы
	success := p.executeJob(ctx, job)

	// Обновляем execution с результатом
	finishedAt := time.Now().UnixMilli()
	if success {
		execution.Status = "completed"
	} else {
		execution.Status = "failed"
	}
	execution.FinishedAt = finishedAt

	// Отправляем результат обратно в планировщик
	if err := p.publisher.Publish(ctx, execution); err != nil {
		p.logger.Error("Failed to publish execution result",
			zap.String("execution_id", executionID),
			zap.String("job_id", job.ID),
			zap.Error(err))
		return fmt.Errorf("publish execution result: %w", err)
	}

	p.logger.Info("Job processed successfully",
		zap.String("job_id", job.ID),
		zap.String("execution_id", executionID),
		zap.String("status", execution.Status),
		zap.Int64("duration_ms", finishedAt-startedAt))

	return nil
}

// executeJob выполняет джобу
// В реальном приложении здесь будет конкретная логика выполнения
func (p *JobProcessor) executeJob(ctx context.Context, job *dto.JobDTO) bool {
	// Симуляция обработки джобы
	// В реальном приложении здесь может быть:
	// - Вызов внешнего API
	// - Обработка данных
	// - Выполнение скрипта
	// - и т.д.

	p.logger.Debug("Executing job",
		zap.String("job_id", job.ID),
		zap.Any("payload", job.Payload))

	// Симулируем работу (например, обработка payload)
	// В реальном приложении здесь будет реальная логика
	select {
	case <-time.After(1 * time.Second): // Симуляция работы
		// В реальном приложении здесь будет обработка
		p.logger.Debug("Job execution completed",
			zap.String("job_id", job.ID))
		return true
	case <-ctx.Done():
		p.logger.Warn("Job execution cancelled",
			zap.String("job_id", job.ID))
		return false
	}
}
