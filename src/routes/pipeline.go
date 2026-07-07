package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/saradab-mindfire/data-processing-pipeline/src/controllers"
)

func SetupRoutes(router *gin.Engine) {
	router.Static("/exports", "exports")

	v1 := router.Group("/api/v1")
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
