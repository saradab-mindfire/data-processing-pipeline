package stage

import (
	"context"
	"strings"
	"sync"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

func Transform(ctx context.Context, wg *sync.WaitGroup, in <-chan shared.Record, out chan<- shared.Record) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case record, ok := <-in:
			if !ok {
				return
			}

			for key, value := range record.Fields {
				record.Fields[key] = strings.TrimSpace(value)
			}

			select {
			case <-ctx.Done():
				return
			case out <- record:
			}
		}
	}
}
