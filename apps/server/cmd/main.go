package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/server/routes"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/config"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/workerclient"
)

func main() {
	fmt.Println("Server is running.")

	cfg := config.Load()

	database.Connect(cfg.DATABASEURL())
	database.Migrate()
	workerclient.Init(cfg.WorkerURL)

	router := gin.Default()

	routes.SetupRoutes(router, cfg.APIKey)

	router.Run(cfg.ServerAddr)
}
