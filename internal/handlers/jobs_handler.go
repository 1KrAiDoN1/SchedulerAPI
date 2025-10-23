package handlers

import (
	"log/slog"
	"scheduler/internal/services"

	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	log        *slog.Logger
	jobService services.JobServiceInterface
}

func NewJobHandler(logger *slog.Logger, jobService services.JobServiceInterface) *JobHandler {
	return &JobHandler{
		log:        logger,
		jobService: jobService,
	}
}

func (h *JobHandler) CreateJob(c *gin.Context) {
	c.JSON(201, gin.H{"message": "CreateJob not implemented yet"})
}

func (h *JobHandler) GetJobs(c *gin.Context) {
	c.JSON(200, gin.H{"message": "GetJobs not implemented yet"})
}

func (h *JobHandler) GetJobByID(c *gin.Context) {
	c.JSON(200, gin.H{"message": "GetJobs not implemented yet"})
}

func (h *JobHandler) DeleteJob(c *gin.Context) {
	c.JSON(200, gin.H{"message": "GetJobs not implemented yet"})
}

func (h *JobHandler) GetJobExecutions(c *gin.Context) {
	c.JSON(200, gin.H{"message": "GetJobs not implemented yet"})
}
