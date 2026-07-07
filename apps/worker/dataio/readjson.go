package dataio

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

func ReadJSON(ctx context.Context, wg *sync.WaitGroup, pipelineID, path, recordsPath string, out chan<- shared.Record) {
	defer wg.Done() // tells run() this reader is done (see readWg)

	body, err := openSource(path)
	if err != nil {
		shared.SaveError(pipelineID, "could not open "+path+": "+err.Error())
		return
	}
	defer body.Close()

	var data any
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		shared.SaveError(pipelineID, "could not decode JSON from "+path+": "+err.Error())
		return
	}

	if recordsPath != "" {
		obj, ok := data.(map[string]any)
		if !ok {
			shared.SaveError(pipelineID, "records_path is set but "+path+" is not a JSON object")
			return
		}
		data, ok = obj[recordsPath]
		if !ok {
			shared.SaveError(pipelineID, "records_path \""+recordsPath+"\" not found in "+path)
			return
		}
	}

	items, ok := data.([]any)
	if !ok {
		items = []any{data} // a single JSON object -> one record
	}

	for _, item := range items {
		fields, ok := item.(map[string]any)
		if !ok {
			shared.SaveError(pipelineID, "skipping non-object element in "+path)
			continue
		}

		record := shared.Record{ID: uuid.NewString(), Fields: make(map[string]string, len(fields))}
		for key, value := range fields {
			record.Fields[key] = jsonValueToString(value)
		}

		select {
		case <-ctx.Done(): // pipeline was cancelled, stop reading immediately
			return
		case out <- record:
		}
	}
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
