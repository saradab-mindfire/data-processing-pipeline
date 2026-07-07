package pipelines

import (
	"context"
	"encoding/csv"
	"sync"
	"fmt"

	"github.com/google/uuid"
)

func readCSV(ctx context.Context, wg *sync.WaitGroup, pipelineID, path string, out chan<- Record) {
	defer wg.Done() // tells run() this reader is done 

	file, err := openSource(path)
	if err != nil {
		saveError(pipelineID, "could not open "+path+": "+err.Error())
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		saveError(pipelineID, "could not read header row in "+path+": "+err.Error())
		return
	}

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
			case out <- Record{ID: uuid.NewString(), Fields: fields}:
		}

		fmt.Println(row, "XCY")
	}
}