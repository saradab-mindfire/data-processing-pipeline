package pipelines

import (
	"encoding/json"
	"sync"
	"time"
	"fmt"

	"github.com/saradab-mindfire/data-processing-pipeline/src/database"
	"github.com/saradab-mindfire/data-processing-pipeline/src/models"
)

func run(j *job, pipelineID string, req PipelineRequest) {
	recordsCh := make(chan Record, 100)
	validatedCh := make(chan Record, 100)
	transformedCh := make(chan Record, 100)

	var readWg sync.WaitGroup
	for _, source := range req.Sources {
		readWg.Add(1)
		switch source.Type {
		case "csv":
			go readCSV(j.ctx, &readWg, pipelineID, source.Path, recordsCh)
		case "json":
			fmt.Println("Coming Here")
			go readJSON(j.ctx, &readWg, pipelineID, source.Path, source.RecordsPath, recordsCh)
		default:
			readWg.Done()
			saveError(pipelineID, "source type not supported yet: "+source.Type)
		}
	}
	go func() {
		readWg.Wait() 
		close(recordsCh)
	}()

	var validateWg sync.WaitGroup
	for i := 0; i < validationWorkers; i++ {
		validateWg.Add(1)
		go validate(j, &validateWg, pipelineID, recordsCh, validatedCh)
	}
	go func() {
		validateWg.Wait()
		close(validatedCh)
	}()

	var transformWg sync.WaitGroup
	for i := 0; i < transformWorkers; i++ {
		transformWg.Add(1)
		go transform(j.ctx, &transformWg, validatedCh, transformedCh)
	}
	go func() {
		transformWg.Wait() 
		close(transformedCh)
	}()

	records := make([]Record, 0)
	for record := range transformedCh {
		records = append(records, record)
	}

	exportResult(pipelineID, records)

	status := models.StatusCompleted
	if j.ctx.Err() != nil { 
		status = models.StatusCancelled
	}

	processed, valid, invalid := j.counts()
	database.Instance.Model(&models.Pipeline{}).Where("id = ?", pipelineID).Updates(map[string]any{
		"STATUS":            status,
		"COMPLETED_AT":      time.Now(),
		"TOTAL_RECORDS":     processed, 
		"PROCESSED_RECORDS": processed,
		"VALID_RECORDS":     valid,
		"INVALID_RECORDS":   invalid,
	})

	removeJob(pipelineID)
}

func jsonValueToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}