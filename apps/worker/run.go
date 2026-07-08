package worker

import (
	"context"
	"sync"
	"time"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/dataio"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/stage"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/queue"
)

const progressFlushInterval = 500 * time.Millisecond

func run(j *job, pipelineID string, req models.PipelineRequest) {
	claim := database.Instance.Model(&models.Pipeline{}).
		Where("id = ? AND status = ?", pipelineID, models.StatusPending).
		Update("status", models.StatusProcessing)
	if claim.Error != nil {
		shared.SaveError(pipelineID, "could not claim pipeline: "+claim.Error.Error())
		removeJob(pipelineID)
		return
	}
	if claim.RowsAffected == 0 {
		removeJob(pipelineID)
		return
	}

	flushCtx, stopFlush := context.WithCancel(context.Background())
	flushDone := make(chan struct{})
	go func() {
		defer close(flushDone)
		ticker := time.NewTicker(progressFlushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				processed, valid, invalid := j.counts()
				database.Instance.Model(&models.Pipeline{}).Where("id = ?", pipelineID).Updates(map[string]any{
					"processed_records": processed,
					"valid_records":     valid,
					"invalid_records":   invalid,
				})
			case <-flushCtx.Done():
				return
			}
		}
	}()

	recordsCh := make(chan shared.Record, 100)
	validatedCh := make(chan shared.Record, 100)
	transformedCh := make(chan shared.Record, 100)

	var readWg sync.WaitGroup
	for _, source := range req.Sources {
		readWg.Add(1)
		switch source.Type {
		case "csv":
			go dataio.ReadCSV(j.ctx, &readWg, pipelineID, source.Path, recordsCh)
		case "json":
			go dataio.ReadJSON(j.ctx, &readWg, pipelineID, source.Path, source.RecordsPath, recordsCh)
		default:
			readWg.Done()
			shared.SaveError(pipelineID, "source type not supported yet: "+source.Type)
		}
	}
	go func() {
		readWg.Wait()
		close(recordsCh)
	}()

	var validateWg sync.WaitGroup
	for i := 0; i < validationWorkers; i++ {
		validateWg.Add(1)
		go stage.Validate(j.ctx, &validateWg, pipelineID, j.addValid, j.addInvalid, recordsCh, validatedCh)
	}
	go func() {
		validateWg.Wait()
		close(validatedCh)
	}()

	var transformWg sync.WaitGroup
	for i := 0; i < transformWorkers; i++ {
		transformWg.Add(1)
		go stage.Transform(j.ctx, &transformWg, validatedCh, transformedCh)
	}
	go func() {
		transformWg.Wait()
		close(transformedCh)
	}()

	records := make([]shared.Record, 0)
	for record := range transformedCh {
		records = append(records, record)
	}

	stopFlush()
	<-flushDone

	status := models.StatusCompleted
	if j.ctx.Err() != nil {
		status = models.StatusCancelled
	} else {
		dataio.ExportResult(pipelineID, records)
	}

	processed, valid, invalid := j.counts()
	database.Instance.Model(&models.Pipeline{}).Where("id = ?", pipelineID).Updates(map[string]any{
		"status":            status,
		"completed_at":      time.Now(),
		"total_records":     processed,
		"processed_records": processed,
		"valid_records":     valid,
		"invalid_records":   invalid,
	})

	queue.DeleteProgress(pipelineID)
	removeJob(pipelineID)
}
