package pipelines

import (
	"strings"
	"sync"
)

func validate(j *job, wg *sync.WaitGroup, pipelineID string, in <-chan Record, out chan<- Record) {
	defer wg.Done() // tells run() this worker is done (see validateWg)

	for {
		select {
		case <-j.ctx.Done():
			return
		case record, ok := <-in:
			if !ok { 
				return
			}

			if !hasNonEmptyField(record) {
				j.addInvalid()
				saveError(pipelineID, "record "+record.ID+": all fields are empty")
				continue
			}
			j.addValid()

			select {
			case <-j.ctx.Done():
				return
			case out <- record:
			}
		}
	}
}

func hasNonEmptyField(record Record) bool {
	for _, value := range record.Fields {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
