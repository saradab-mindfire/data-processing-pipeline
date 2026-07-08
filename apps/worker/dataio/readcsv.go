package dataio

import (
	"context"
	"encoding/csv"
	"sync"

	"github.com/google/uuid"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

func ReadCSV(ctx context.Context, wg *sync.WaitGroup, pipelineID, path string, out chan<- shared.Record) {
	defer wg.Done() // tells run() this reader is done

	file, err := openSource(path)
	if err != nil {
		shared.SaveError(pipelineID, "could not open "+path+": "+err.Error())
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.ReuseRecord = true
	headerRow, err := reader.Read()
	if err != nil {
		shared.SaveError(pipelineID, "could not read header row in "+path+": "+err.Error())
		return
	}
	headers := append([]string(nil), headerRow...)

	for {
		row, err := reader.Read()
		if err != nil {
			return
		}

		fields := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(row) {
				fields[header] = row[i]
			}
		}

		select {
		case <-ctx.Done(): // pipeline was cancelled, stop reading immediately
			return
		case out <- shared.Record{ID: uuid.NewString(), Fields: fields}:
		}
	}
}
