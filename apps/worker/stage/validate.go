package stage

import (
	"context"
	"strings"
	"sync"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

func Validate(ctx context.Context, wg *sync.WaitGroup, pipelineID string, addValid, addInvalid func(), in <-chan shared.Record, out chan<- shared.Record) {
	defer wg.Done() 

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
				shared.SaveError(pipelineID, "record "+record.ID+": all fields are empty")
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
