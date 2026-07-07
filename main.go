package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/config"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/server/routes"
)

func main() {
	fmt.Println("Server is running.")

	cfg := config.Load()

	database.Connect(cfg.DATABASEURL())
	database.Migrate()

	router := gin.Default()

	routes.SetupRoutes(router, cfg.APIKey)

	router.Run(cfg.ServerAddr)
}