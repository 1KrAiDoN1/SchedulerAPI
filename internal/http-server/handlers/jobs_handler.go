package handlers

import (
	"context"
	"net/http"
	"scheduler/internal/domain/entity"
	"scheduler/internal/models/dto"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type JobsHandler struct {
	log        *zap.Logger
	jobService JobServiceInterface
}

func NewJobsHandler(logger *zap.Logger, jobService JobServiceInterface) *JobsHandler {
	return &JobsHandler{
		log:        logger,
		jobService: jobService,
	}
}

type JobServiceInterface interface {
	CreateJob(ctx context.Context, job *entity.Job) (string, error)
	GetJob(ctx context.Context, jobID string) (*entity.Job, error)
	GetJobs(ctx context.Context) ([]*entity.Job, error)
	DeleteJob(ctx context.Context, jobID string) error
	GetJobExecutions(ctx context.Context, jobID string) ([]*entity.Execution, error)
	CompleteJob(ctx context.Context, jobID string, success bool) error
	HandleExecutionResult(ctx context.Context, execution *entity.Execution) error
	Start(ctx context.Context) error
}

func (h *JobsHandler) CreateJob(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req dto.JobCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error("Failed to bind JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	job := &entity.Job{
		Status:         entity.StatusQueued,
		LastFinishedAt: 0,
		Payload:        req.Payload,
	}

	// Определяем тип задачи и парсим параметры
	if req.Once != "" {
		onceTime, err := time.Parse(time.RFC3339, req.Once)
		if err != nil {
			h.log.Error("Failed to parse once time", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid once time format. Use RFC3339 format"})
			return
		}
		onceTimestamp := onceTime.UnixMilli()
		job.Kind = entity.JobKindOnce
		job.Once = &onceTimestamp
	} else if req.Interval != "" {
		interval, err := time.ParseDuration(req.Interval)
		if err != nil {
			h.log.Error("Failed to parse interval", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval format. Use Go duration format (e.g., 1m, 1h, 30s)"})
			return
		}
		job.Kind = entity.JobKindInterval
		job.Interval = &interval
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either 'once' or 'interval' must be provided"})
		return
	}

	jobID, err := h.jobService.CreateJob(ctx, job)
	if err != nil {
		h.log.Error("Failed to create job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": jobID})
}

func (h *JobsHandler) GetJobs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	jobs, err := h.jobService.GetJobs(ctx)
	if err != nil {
		h.log.Error("Failed to get jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get jobs"})
		return
	}

	// Конвертируем entity.Job в dto.JobDTO
	jobDTOs := make([]dto.JobDTO, 0, len(jobs))
	for _, job := range jobs {
		jobDTO := h.entityToDTO(job)
		jobDTOs = append(jobDTOs, jobDTO)
	}

	c.JSON(http.StatusOK, jobDTOs)
}

func (h *JobsHandler) GetJobByID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	job, err := h.jobService.GetJob(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", zap.Error(err), zap.String("job_id", jobID))
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	jobDTO := h.entityToDTO(job)
	c.JSON(http.StatusOK, jobDTO)
}

func (h *JobsHandler) DeleteJob(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	err := h.jobService.DeleteJob(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to delete job", zap.Error(err), zap.String("job_id", jobID))
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

func (h *JobsHandler) GetJobExecutions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	executions, err := h.jobService.GetJobExecutions(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job executions", zap.Error(err), zap.String("job_id", jobID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job executions"})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// entityToDTO конвертирует entity.Job в dto.JobDTO
func (h *JobsHandler) entityToDTO(job *entity.Job) dto.JobDTO {
	jobDTO := dto.JobDTO{
		ID:             job.ID,
		Kind:           int(job.Kind),
		Status:         string(job.Status),
		LastFinishedAt: job.LastFinishedAt,
		Payload:        job.Payload,
	}

	if job.Interval != nil {
		intervalStr := job.Interval.String()
		jobDTO.Interval = &intervalStr
	}

	if job.Once != nil {
		jobDTO.Once = job.Once
	}

	return jobDTO
}
