package worker

import (
	"sync"
	"time"

	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/dataio"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/stage"
)

func run(j *job, pipelineID string, req PipelineRequest) {
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

	dataio.ExportResult(pipelineID, records)

	status := models.StatusCompleted
	if j.ctx.Err() != nil {
		status = models.StatusCancelled
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

	removeJob(pipelineID)
}
