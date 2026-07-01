package controllers

import (
	"net/http"
	"time"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/saradab-mindfire/data-processing-pipeline/src/database"
	"github.com/saradab-mindfire/data-processing-pipeline/src/models"
)

func CreatePipeline(c *gin.Context) {
	var pipeline models.Pipeline

	fmt.Println("Received request to create pipeline:", c.Request.Body)

	if err := c.ShouldBindJSON(&pipeline); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.Instance.Create(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, pipeline)
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
	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	var percentage float64
	if pipeline.TOTAL_RECORDS > 0 {
		percentage = float64(pipeline.PROCESSED_RECORDS) / float64(pipeline.TOTAL_RECORDS) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                pipeline.ID,
		"status":            pipeline.STATUS,
		"total_records":     pipeline.TOTAL_RECORDS,
		"processed_records": pipeline.PROCESSED_RECORDS,
		"valid_records":     pipeline.VALID_RECORDS,
		"invalid_records":   pipeline.INVALID_RECORDS,
		"percentage":        percentage,
		"started_at":        pipeline.STARTED_AT,
		"completed_at":      pipeline.COMPLETED_AT,
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

	c.JSON(http.StatusOK, gin.H{
		"id":                pipeline.ID,
		"status":            pipeline.STATUS,
		"total_records":     pipeline.TOTAL_RECORDS,
		"processed_records": pipeline.PROCESSED_RECORDS,
		"valid_records":     pipeline.VALID_RECORDS,
		"invalid_records":   pipeline.INVALID_RECORDS,
		"completed_at":      pipeline.COMPLETED_AT,
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
	var pipeline models.Pipeline
	if err := database.Instance.First(&pipeline, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	if pipeline.STATUS == models.StatusCompleted || pipeline.STATUS == models.StatusCancelled || pipeline.STATUS == models.StatusFailed {
		c.JSON(http.StatusConflict, gin.H{"error": "pipeline cannot be cancelled from status: " + string(pipeline.STATUS)})
		return
	}

	pipeline.STATUS = models.StatusCancelled
	pipeline.COMPLETED_AT = time.Now()
	if err := database.Instance.Save(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipeline)
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
