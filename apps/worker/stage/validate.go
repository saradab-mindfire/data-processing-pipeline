package stage

import (
	"context"
	"strings"
	"sync"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

const errorBatchSize = 50

func Validate(ctx context.Context, wg *sync.WaitGroup, pipelineID string, addValid, addInvalid func(), in <-chan shared.Record, out chan<- shared.Record) {
	defer wg.Done()

	errBatch := make([]string, 0, errorBatchSize)
	flushErrors := func() {
		if len(errBatch) == 0 {
			return
		}
		shared.SaveErrors(pipelineID, errBatch)
		errBatch = errBatch[:0]
	}
	defer flushErrors()

	for {
		select {
		case <-ctx.Done():
			return
		case record, ok := <-in:
			if !ok {
				return
			}

			if !hasNonEmptyField(record) {
				addInvalid()
				errBatch = append(errBatch, "record "+record.ID+": all fields are empty")
				if len(errBatch) >= errorBatchSize {
					flushErrors()
				}
				continue
			}
			addValid()

			select {
			case <-ctx.Done():
				return
			case out <- record:
			}
		}
	}
}

func hasNonEmptyField(record shared.Record) bool {
	for _, value := range record.Fields {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
