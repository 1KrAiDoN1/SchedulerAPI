package services

import (
	"context"
	"errors"
	"fmt"
	"scheduler/internal/domain/entity"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrJobNotFound = errors.New("job not found")
)

type JobsService struct {
	jobsRepo       JobRepositoryInterface
	executionsRepo ExecutionRepositoryInterface
	publisher      JobPublisherInterface
	running        map[string]*entity.RunningJob
	runningMu      sync.RWMutex // Мьютекс для защиты map running от конкурентного доступа
	interval       time.Duration
	logger         *zap.Logger
}

func NewJobsService(jobsRepo JobRepositoryInterface, executionsRepo ExecutionRepositoryInterface, publisher JobPublisherInterface, interval time.Duration, logger *zap.Logger) *JobsService {
	return &JobsService{
		jobsRepo:       jobsRepo,
		executionsRepo: executionsRepo,
		publisher:      publisher,
		running:        make(map[string]*entity.RunningJob),
		runningMu:      sync.RWMutex{},
		interval:       interval,
		logger:         logger,
	}
}

type JobRepositoryInterface interface {
	Create(ctx context.Context, job *entity.Job) error
	Read(ctx context.Context, jobID string) (*entity.Job, error)
	List(ctx context.Context) ([]*entity.Job, error)
	Upsert(ctx context.Context, jobs []*entity.Job) error
	Delete(ctx context.Context, jobID string) error
}

type JobPublisherInterface interface {
	Publish(ctx context.Context, job *entity.Job) error
}

type JobSubscriberInterface interface {
	Subscribe(ctx context.Context, handler func(ctx context.Context, execution *entity.Execution) error) error
	Close() error
}

type ExecutionRepositoryInterface interface {
	Create(ctx context.Context, execution *entity.Execution) error
	GetByJobID(ctx context.Context, jobID string) ([]*entity.Execution, error)
}

// CreateJob создает новую задачу в репозитории
func (j *JobsService) CreateJob(ctx context.Context, job *entity.Job) (string, error) {
	job.ID = uuid.NewString() //присваиваем ID
	err := j.jobsRepo.Create(ctx, job)
	if err != nil {
		return "", fmt.Errorf("create job: %w", err)
	}

	return job.ID, nil
}

// Start запускает бесконечный цикл планировщика с периодической проверкой задач
// При получении сигнала отмены контекста корректно завершает работу
func (j *JobsService) Start(ctx context.Context) error {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Каждый интервал запускаем метод tick для проверки задач
			if err := j.tick(ctx); err != nil {
				return fmt.Errorf("tick: %w", err)
			}
		case <-ctx.Done():
			// При отмене контекста завершаем работу
			j.logger.Info("stopping jobs service", zap.Error(ctx.Err()))
			return ctx.Err()
		}
	}
}

// tick проверяет все задачи и запускает те, которые должны быть выполнены
func (j *JobsService) tick(ctx context.Context) error {
	// Получаем все задачи из репозитория
	jobs, err := j.jobsRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list jobs: %w", err)
	}

	// Создаем мапу задач из репозитория для быстрого поиска
	repoJobs := make(map[string]*entity.Job, len(jobs))
	for i := range jobs {
		repoJobs[jobs[i].ID] = jobs[i]
	}

	// Проверяем запущенные задачи: если задача удалена из репозитория,
	// отменяем её выполнение и удаляем из списка запущенных
	j.runningMu.Lock()
	for jobID, runningJob := range j.running {
		if _, existsInRepo := repoJobs[jobID]; !existsInRepo {
			j.logger.Debug("stopping removed job", zap.String("job_id", jobID))
			runningJob.Cancel()
			delete(j.running, jobID)
		}
	}
	j.runningMu.Unlock()

	// Определяем текущее время в миллисекундах
	now := time.Now().UnixMilli()

	// Список задач, которые нужно обновить в репозитории
	var updates []*entity.Job

	// Проходим по всем задачам из репозитория
	for jobID, job := range repoJobs {
		// Пропускаем задачи, которые уже выполняются
		j.runningMu.RLock()
		_, isRunning := j.running[jobID]
		j.runningMu.RUnlock()

		if isRunning {
			j.logger.Debug("skipping already running job", zap.String("job_id", jobID))
			continue
		}

		// Проверяем, нужно ли запускать задачу в зависимости от её типа
		shouldRun := false

		switch job.Kind {
		case entity.JobKindInterval:
			// Для периодических задач проверяем, что интервал прошел с последнего завершения
			if job.Interval == nil {
				j.logger.Warn("interval job has nil interval", zap.String("job_id", jobID))
				continue
			}

			// Вычисляем время следующего запуска
			nextRunTime := job.LastFinishedAt + job.Interval.Milliseconds()
			if now >= nextRunTime {
				shouldRun = true
			}

		case entity.JobKindOnce:
			// Для одноразовых задач проверяем, что они еще не выполнялись
			// и время выполнения наступило
			if job.Once == nil {
				j.logger.Warn("once job has nil once time", zap.String("job_id", jobID))
				continue
			}

			// Задача должна выполниться один раз, если она еще не выполнялась
			// и время выполнения уже наступило
			if job.LastFinishedAt == 0 && now >= *job.Once {
				shouldRun = true
			}

		default:
			j.logger.Warn("unknown job kind", zap.String("job_id", jobID), zap.Uint8("kind", uint8(job.Kind)))
			continue
		}

		// Если задача должна быть запущена, запускаем её в отдельной горутине
		if shouldRun {
			// Копируем задачу для передачи в горутину
			jobCopy := *job
			jobCopy.Status = entity.StatusQueued

			// Запускаем задачу асинхронно
			go j.runJob(ctx, &jobCopy)

			// Добавляем задачу в список обновлений с статусом Queued
			updates = append(updates, &jobCopy)

			j.logger.Info("scheduled job for execution",
				zap.String("job_id", jobID),
				zap.String("kind", string(job.Kind)),
				zap.String("status", string(entity.StatusQueued)))
		}
	}

	// Обновляем статусы запущенных задач в репозитории
	if len(updates) > 0 {
		if err := j.jobsRepo.Upsert(ctx, updates); err != nil {
			return fmt.Errorf("upsert scheduled jobs: %w", err)
		}
	}

	return nil
}

// runJob выполняет задачу: публикует её через publisher и управляет её жизненным циклом
func (j *JobsService) runJob(ctx context.Context, job *entity.Job) {
	// Создаем контекст с возможностью отмены для этой задачи
	jobCtx, cancel := context.WithCancel(ctx)

	// Создаем структуру запущенной задачи
	runningJob := &entity.RunningJob{
		Job:    job,
		Cancel: cancel,
	}

	// Добавляем задачу в список выполняющихся задач
	j.runningMu.Lock()
	// Проверяем, не запущена ли задача уже (на случай race condition)
	if _, exists := j.running[job.ID]; exists {
		j.runningMu.Unlock()
		j.logger.Warn("job already running, skipping", zap.String("job_id", job.ID))
		cancel()
		return
	}
	j.running[job.ID] = runningJob
	j.runningMu.Unlock()

	// Устанавливаем статус задачи как выполняющейся
	job.Status = entity.StatusRunning

	// Обновляем статус в репозитории
	if err := j.jobsRepo.Upsert(ctx, []*entity.Job{job}); err != nil {
		j.logger.Error("failed to update job status to running",
			zap.String("job_id", job.ID),
			zap.Error(err))
	}

	// Публикуем задачу через publisher, если он настроен
	if j.publisher != nil {
		if err := j.publisher.Publish(jobCtx, job); err != nil {
			j.logger.Error("failed to publish job",
				zap.String("job_id", job.ID),
				zap.Error(err))

			// Обновляем статус на Failed при ошибке публикации
			job.Status = entity.StatusFailed
			if err := j.jobsRepo.Upsert(ctx, []*entity.Job{job}); err != nil {
				j.logger.Error("failed to update job status to failed",
					zap.String("job_id", job.ID),
					zap.Error(err))
			}

			// Удаляем задачу из списка выполняющихся
			j.runningMu.Lock()
			delete(j.running, job.ID)
			j.runningMu.Unlock()
			cancel()
			return
		}

		j.logger.Info("job published successfully",
			zap.String("job_id", job.ID),
			zap.String("kind", string(job.Kind)))
	} else {
		j.runningMu.Lock()
		delete(j.running, job.ID)
		j.runningMu.Unlock()
		cancel()
	}
}

// CompleteJob обновляет задачу после её успешного завершения
func (j *JobsService) CompleteJob(ctx context.Context, jobID string, success bool) error {
	j.runningMu.Lock()
	runningJob, exists := j.running[jobID]
	if !exists {
		j.runningMu.Unlock()
		// Если задача не в running, пытаемся обновить её напрямую из БД
		//ищем в бд по id
		job, err := j.jobsRepo.Read(ctx, jobID)
		if err != nil {
			return fmt.Errorf("job %s is not running and not found in DB: %w", jobID, err)
		}
		// Обновляем задачу из БД
		job.LastFinishedAt = time.Now().UnixMilli()
		if success {
			job.Status = entity.StatusCompleted
		} else {
			job.Status = entity.StatusFailed
		}
		if err := j.jobsRepo.Upsert(ctx, []*entity.Job{job}); err != nil {
			return fmt.Errorf("failed to update completed job: %w", err)
		}
		j.logger.Info("job completed (was not in running map)",
			zap.String("job_id", jobID),
			zap.Bool("success", success),
			zap.String("status", string(job.Status)))
		return nil
	}
	delete(j.running, jobID)
	j.runningMu.Unlock()

	// Обновляем время последнего завершения
	runningJob.Job.LastFinishedAt = time.Now().UnixMilli()

	// Устанавливаем статус в зависимости от результата
	if success {
		runningJob.Job.Status = entity.StatusCompleted
	} else {
		runningJob.Job.Status = entity.StatusFailed
	}

	// Обновляем задачу в репозитории
	if err := j.jobsRepo.Upsert(ctx, []*entity.Job{runningJob.Job}); err != nil {
		return fmt.Errorf("failed to update completed job: %w", err)
	}

	// Отменяем контекст задачи
	runningJob.Cancel()

	j.logger.Info("job completed",
		zap.String("job_id", jobID),
		zap.Bool("success", success),
		zap.String("status", string(runningJob.Job.Status)))

	return nil
}

// GetJob получает задачу по ID
func (j *JobsService) GetJob(ctx context.Context, jobID string) (*entity.Job, error) {
	job, err := j.jobsRepo.Read(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return job, nil
}

// GetJobs получает все задачи
func (j *JobsService) GetJobs(ctx context.Context) ([]*entity.Job, error) {
	jobs, err := j.jobsRepo.List(ctx)
	if err != nil {
		if err == ErrJobNotFound {
			return []*entity.Job{}, nil
		}
		return nil, fmt.Errorf("get jobs: %w", err)
	}
	return jobs, nil
}

// DeleteJob удаляет задачу по ID
func (j *JobsService) DeleteJob(ctx context.Context, jobID string) error {
	// Отменяем задачу, если она выполняется
	j.runningMu.Lock()
	if runningJob, exists := j.running[jobID]; exists {
		runningJob.Cancel()
		delete(j.running, jobID)
	}
	j.runningMu.Unlock()

	err := j.jobsRepo.Delete(ctx, jobID)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	return nil
}

// GetJobExecutions получает все выполнения задачи по jobID
func (j *JobsService) GetJobExecutions(ctx context.Context, jobID string) ([]*entity.Execution, error) {
	executions, err := j.executionsRepo.GetByJobID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get job executions: %w", err)
	}
	return executions, nil
}

// HandleExecutionResult обрабатывает результат выполнения задачи от воркера
func (j *JobsService) HandleExecutionResult(ctx context.Context, execution *entity.Execution) error {
	if err := j.executionsRepo.Create(ctx, execution); err != nil {
		j.logger.Error("failed to create execution",
			zap.String("execution_id", execution.ID),
			zap.String("job_id", execution.JobID),
			zap.Error(err))
		return fmt.Errorf("create execution: %w", err)
	}

	// Определяем успешность выполнения на основе статуса
	success := execution.Status == "completed" || execution.Status == "success"

	// Обновляем задачу через CompleteJob
	if err := j.CompleteJob(ctx, execution.JobID, success); err != nil {
		j.logger.Warn("failed to complete job (may be already completed or deleted)",
			zap.String("job_id", execution.JobID),
			zap.String("execution_id", execution.ID),
			zap.Error(err))
		// Не возвращаем ошибку, так как execution уже сохранен
		// Это может произойти, если задача была удалена или уже завершена
	}

	j.logger.Info("execution result processed",
		zap.String("execution_id", execution.ID),
		zap.String("job_id", execution.JobID),
		zap.String("status", execution.Status),
		zap.String("worker_id", execution.WorkerID))

	return nil
}
