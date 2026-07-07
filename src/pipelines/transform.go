package pipelines

import (
	"context"
	"strings"
	"sync"
)

func transform(ctx context.Context, wg *sync.WaitGroup, in <-chan Record, out chan<- Record) {
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
