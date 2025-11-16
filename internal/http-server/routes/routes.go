package routes

import (
	"scheduler/internal/domain/handler"

	"github.com/gin-gonic/gin"
)

func SetupJobsRoutes(router *gin.RouterGroup, jobHandler handler.JobsHandlerInterface) {
	jobs := router.Group("/jobs")
	{
		jobs.POST("", jobHandler.CreateJob)
		jobs.GET("", jobHandler.GetJobs)
		jobs.GET("/:job_id", jobHandler.GetJobByID)
		jobs.DELETE("/:job_id", jobHandler.DeleteJob)
		jobs.GET("/:job_id/executions", jobHandler.GetJobExecutions)
	}
}
