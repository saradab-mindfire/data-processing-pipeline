package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/server/controllers"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/server/middleware"
)

const apiV1Prefix = "/api/v1"

func SetupRoutes(router *gin.Engine, apiKey string) {
	router.Use(middleware.RequireAPIKey(apiKey))
	router.Use(middleware.RateLimit())

	router.Static("/exports", "exports")

	v1 := router.Group(apiV1Prefix)
	pipelines := v1.Group("/pipelines")
	{
		pipelines.POST("", controllers.CreatePipeline)
		pipelines.GET("", controllers.GetPipelines)
		pipelines.GET("/:id", controllers.GetPipeline)
		pipelines.GET("/:id/progress", controllers.GetPipelineProgress)
		pipelines.GET("/:id/results", controllers.GetPipelineResults)
		pipelines.GET("/:id/errors", controllers.GetPipelineErrors)
		pipelines.PATCH("/:id/cancel", controllers.CancelPipeline)
		pipelines.PUT("/:id", controllers.UpdatePipeline)
		pipelines.DELETE("/:id", controllers.DeletePipeline)
	}
}
