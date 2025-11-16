package handler

import "github.com/gin-gonic/gin"

type JobsHandlerInterface interface {
	CreateJob(c *gin.Context)
	GetJobs(c *gin.Context)
	GetJobByID(c *gin.Context)
	DeleteJob(c *gin.Context)
	GetJobExecutions(c *gin.Context)
}
