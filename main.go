package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/saradab-mindfire/data-processing-pipeline/src/database"
	"github.com/saradab-mindfire/data-processing-pipeline/src/routes"
)

func main() {
	fmt.Println("Server is running.")

	connectionString := "host=localhost user=admin password=admin123 dbname=data-processing-pipeline port=5432 sslmode=disable"

	database.Connect(connectionString)
	database.Migrate()

	router := gin.Default()

	routes.SetupRoutes(router)

	router.Run("localhost:9090")
}