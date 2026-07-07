package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/saradab-mindfire/data-processing-pipeline/src/database"
	"github.com/saradab-mindfire/data-processing-pipeline/src/models"
	"github.com/saradab-mindfire/data-processing-pipeline/src/pipelines"
)

func CreatePipeline(c *gin.Context) {
	var req pipelines.PipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbPipeline := models.Pipeline{
		ID:         uuid.NewString(),
		NAME:       "pipeline-" + time.Now().Format("20060102-150405"),
		STATUS:     models.StatusProcessing,
		STARTED_AT: time.Now(),
	}

	if err := database.Instance.Create(&dbPipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	pipelines.Start(dbPipeline.ID, req)

	c.JSON(http.StatusAccepted, dbPipeline)
}

func GetPipelines(c *gin.Context) {
	var pipelines []models.Pipeline
	if err := database.Instance.Find(&pipelines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipelines)
}

func GetPipeline(c *gin.Context) {
	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

func UpdatePipeline(c *gin.Context) {
	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	var input models.Pipeline
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.ID = pipeline.ID

	if err := database.Instance.Model(&pipeline).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

func GetPipelineProgress(c *gin.Context) {
	var dbPipeline models.Pipeline
	if err := database.Instance.First(&dbPipeline, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	processedRecords := dbPipeline.PROCESSED_RECORDS
	validRecords := dbPipeline.VALID_RECORDS
	invalidRecords := dbPipeline.INVALID_RECORDS

	if processed, valid, invalid, ok := pipelines.Progress(dbPipeline.ID); ok {
		processedRecords = processed
		validRecords = valid
		invalidRecords = invalid
	}

	var percentage float64
	if dbPipeline.TOTAL_RECORDS > 0 {
		percentage = float64(processedRecords) / float64(dbPipeline.TOTAL_RECORDS) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                dbPipeline.ID,
		"status":            dbPipeline.STATUS,
		"total_records":     dbPipeline.TOTAL_RECORDS,
		"processed_records": processedRecords,
		"valid_records":     validRecords,
		"invalid_records":   invalidRecords,
		"percentage":        percentage,
		"started_at":        dbPipeline.STARTED_AT,
		"completed_at":      dbPipeline.COMPLETED_AT,
	})
}

func GetPipelineResults(c *gin.Context) {
	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	if pipeline.STATUS != models.StatusCompleted {
		c.JSON(http.StatusConflict, gin.H{"error": "results are not available until the pipeline has completed"})
		return
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	exportURL := scheme + "://" + c.Request.Host + "/exports/" + pipeline.ID + ".json"

	c.JSON(http.StatusOK, gin.H{
		"id":                pipeline.ID,
		"status":            pipeline.STATUS,
		"total_records":     pipeline.TOTAL_RECORDS,
		"processed_records": pipeline.PROCESSED_RECORDS,
		"valid_records":     pipeline.VALID_RECORDS,
		"invalid_records":   pipeline.INVALID_RECORDS,
		"completed_at":      pipeline.COMPLETED_AT,
		"export_url":        exportURL,
	})
}

func GetPipelineErrors(c *gin.Context) {
	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	var errors []models.PipelineError
	if err := database.Instance.Where("pipeline_id = ?", pipeline.ID).Find(&errors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, errors)
}

func CancelPipeline(c *gin.Context) {
	var dbPipeline models.Pipeline
	if err := database.Instance.First(&dbPipeline, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	if dbPipeline.STATUS == models.StatusCompleted || dbPipeline.STATUS == models.StatusCancelled || dbPipeline.STATUS == models.StatusFailed {
		c.JSON(http.StatusConflict, gin.H{"error": "pipeline cannot be cancelled from status: " + string(dbPipeline.STATUS)})
		return
	}

	if pipelines.Cancel(dbPipeline.ID) {
		// Still running in memory: pipeline.run() will finish writing the
		// final DB status itself.
		c.JSON(http.StatusOK, dbPipeline)
		return
	}

	// No matching in-memory job (e.g. the server restarted since this
	// pipeline started) - fall back to marking it cancelled directly.
	dbPipeline.STATUS = models.StatusCancelled
	dbPipeline.COMPLETED_AT = time.Now()
	if err := database.Instance.Save(&dbPipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dbPipeline)
}

func DeletePipeline(c *gin.Context) {
	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	if err := database.Instance.Delete(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
