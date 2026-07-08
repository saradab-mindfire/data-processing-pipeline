package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/workerclient"
)

const (
	defaultPageLimit = 100
	maxPageLimit     = 500
)

func pageParams(c *gin.Context) (limit, offset int) {
	limit = defaultPageLimit
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v >= 0 {
		offset = v
	}
	return limit, offset
}

// CreatePipeline creates a new pipeline and enqueues it for processing by a worker.
func CreatePipeline(c *gin.Context) {
	var req models.PipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbPipeline := models.Pipeline{
		ID:        uuid.NewString(),
		Name:      "pipeline-" + time.Now().Format("20060102-150405"),
		Status:    models.StatusPending,
		StartedAt: time.Now(),
	}

	if err := database.Instance.Create(&dbPipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := workerclient.Enqueue(c.Request.Context(), dbPipeline.ID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, dbPipeline)
}

// GetPipelines returns all pipelines.
func GetPipelines(c *gin.Context) {
	limit, offset := pageParams(c)

	var pipelines []models.Pipeline
	if err := database.Instance.Order("started_at desc").Limit(limit).Offset(offset).Find(&pipelines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipelines)
}

func parsePipelineID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pipeline id"})
		return "", false
	}
	return id, true
}

// GetPipeline returns a single pipeline by id.
func GetPipeline(c *gin.Context) {
	id, ok := parsePipelineID(c)
	if !ok {
		return
	}

	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

// UpdatePipeline updates fields of an existing pipeline.
func UpdatePipeline(c *gin.Context) {
	id, ok := parsePipelineID(c)
	if !ok {
		return
	}

	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", id).Error; err != nil {
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

// GetPipelineProgress returns live processing progress for a pipeline.
func GetPipelineProgress(c *gin.Context) {
	id, ok := parsePipelineID(c)
	if !ok {
		return
	}

	var dbPipeline models.Pipeline
	if err := database.Instance.First(&dbPipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	processedRecords := dbPipeline.ProcessedRecords
	validRecords := dbPipeline.ValidRecords
	invalidRecords := dbPipeline.InvalidRecords

	if processed, valid, invalid, ok := workerclient.GetProgress(dbPipeline.ID); ok {
		processedRecords = processed
		validRecords = valid
		invalidRecords = invalid
	}

	var percentage float64
	if dbPipeline.TotalRecords > 0 {
		percentage = float64(processedRecords) / float64(dbPipeline.TotalRecords) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                dbPipeline.ID,
		"status":            dbPipeline.Status,
		"total_records":     dbPipeline.TotalRecords,
		"processed_records": processedRecords,
		"valid_records":     validRecords,
		"invalid_records":   invalidRecords,
		"percentage":        percentage,
		"started_at":        dbPipeline.StartedAt,
		"completed_at":      dbPipeline.CompletedAt,
	})
}

// GetPipelineResults returns the results summary and export URL for a completed pipeline.
func GetPipelineResults(c *gin.Context) {
	id, ok := parsePipelineID(c)
	if !ok {
		return
	}

	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	if pipeline.Status != models.StatusCompleted {
		c.JSON(http.StatusConflict, gin.H{"error": "results are not available until the pipeline has completed"})
		return
	}

	exportURL := workerclient.BaseURL() + "/exports/" + pipeline.ID + ".json"

	c.JSON(http.StatusOK, gin.H{
		"id":                pipeline.ID,
		"status":            pipeline.Status,
		"total_records":     pipeline.TotalRecords,
		"processed_records": pipeline.ProcessedRecords,
		"valid_records":     pipeline.ValidRecords,
		"invalid_records":   pipeline.InvalidRecords,
		"completed_at":      pipeline.CompletedAt,
		"export_url":        exportURL,
	})
}

// GetPipelineErrors returns the list of record-level errors recorded for a pipeline.
func GetPipelineErrors(c *gin.Context) {
	id, ok := parsePipelineID(c)
	if !ok {
		return
	}

	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	limit, offset := pageParams(c)

	var errors []models.PipelineError
	if err := database.Instance.Where("pipeline_id = ?", pipeline.ID).Order("created_at desc").Limit(limit).Offset(offset).Find(&errors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, errors)
}

// CancelPipeline cancels a pending or processing pipeline.
func CancelPipeline(c *gin.Context) {
	id, ok := parsePipelineID(c)
	if !ok {
		return
	}

	var dbPipeline models.Pipeline
	if err := database.Instance.First(&dbPipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	if dbPipeline.Status == models.StatusCompleted || dbPipeline.Status == models.StatusCancelled || dbPipeline.Status == models.StatusFailed {
		c.JSON(http.StatusConflict, gin.H{"error": "pipeline cannot be cancelled from status: " + string(dbPipeline.Status)})
		return
	}

	workerclient.Cancel(dbPipeline.ID)

	if dbPipeline.Status == models.StatusProcessing {
		// A worker has already picked this job up; it will observe the
		// cancel notification and move the pipeline to cancelled itself.
		c.JSON(http.StatusOK, dbPipeline)
		return
	}

	dbPipeline.Status = models.StatusCancelled
	dbPipeline.CompletedAt = time.Now()
	if err := database.Instance.Save(&dbPipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dbPipeline)
}

// DeletePipeline deletes a pipeline by id.
func DeletePipeline(c *gin.Context) {
	id, ok := parsePipelineID(c)
	if !ok {
		return
	}

	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	if err := database.Instance.Delete(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
