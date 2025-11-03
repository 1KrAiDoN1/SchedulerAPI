package handlers

import (
	"scheduler/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type JobsHandler struct {
	log        *zap.Logger
	jobService services.JobServiceInterface
}

func NewJobsHandler(logger *zap.Logger, jobService services.JobServiceInterface) *JobsHandler {
	return &JobsHandler{
		log:        logger,
		jobService: jobService,
	}
}

func (h *JobsHandler) CreateJob(c *gin.Context) {
	c.JSON(201, gin.H{"message": "CreateJob not implemented yet"})
}

func (h *JobsHandler) GetJobs(c *gin.Context) {
	c.JSON(200, gin.H{"message": "GetJobs not implemented yet"})
}

func (h *JobsHandler) GetJobByID(c *gin.Context) {
	c.JSON(200, gin.H{"message": "GetJobs not implemented yet"})
}

func (h *JobsHandler) DeleteJob(c *gin.Context) {
	c.JSON(200, gin.H{"message": "GetJobs not implemented yet"})
}

func (h *JobsHandler) GetJobExecutions(c *gin.Context) {
	c.JSON(200, gin.H{"message": "GetJobs not implemented yet"})
}
