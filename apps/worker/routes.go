package worker

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/queue"
)

// SetupRoutes configures routes for internal worker communication
// Redis is avoided to eliminate external resource dependencies
func SetupRoutes(router *gin.Engine) {
	router.Static("/exports", "exports")

	pipelines := router.Group("/internal/pipelines")
	pipelines.POST("", enqueuePipeline)
	pipelines.GET("/:id/progress", getPipelineProgress)
	pipelines.POST("/:id/cancel", cancelPipeline)
}

func enqueuePipeline(c *gin.Context) {
	var body struct {
		PipelineID string                 `json:"pipeline_id" binding:"required"`
		Request    models.PipelineRequest `json:"request"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := queue.Enqueue(c.Request.Context(), body.PipelineID, body.Request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusAccepted)
}

func getPipelineProgress(c *gin.Context) {
	processed, valid, invalid, ok := queue.GetProgress(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "no progress for pipeline"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"processed": processed, "valid": valid, "invalid": invalid})
}

func cancelPipeline(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"cancelled": Cancel(c.Param("id"))})
}
