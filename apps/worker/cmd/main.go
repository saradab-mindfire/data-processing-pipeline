package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/dataio"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/config"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/queue"
)

func main() {
	fmt.Println("Worker is running.")

	cfg := config.Load()

	dataio.InitExportsDir(cfg.ExportsDir)
	database.Connect(cfg.DATABASEURL())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	router := gin.Default()
	worker.SetupRoutes(router, cfg.WorkerInternalToken)
	go func() {
		if err := router.Run(cfg.WorkerAddr); err != nil {
			log.Fatalf("worker: internal API server failed: %v", err)
		}
	}()

	for {
		job, err := queue.Dequeue(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("worker: shutting down")
				return
			}
			log.Println("worker: failed to dequeue job:", err)
			continue
		}
		worker.Start(job.PipelineID, job.Request)
	}
}
