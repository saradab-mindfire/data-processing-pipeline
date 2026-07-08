package dataio

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

	if recordsPath == "" {
		if err := streamJSONRecords(ctx, pipelineID, path, body, out); err != nil {
			shared.SaveError(pipelineID, "could not decode JSON from "+path+": "+err.Error())
		}
		return
	}

	var data any
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		shared.SaveError(pipelineID, "could not decode JSON from "+path+": "+err.Error())
		return
	}

	obj, ok := data.(map[string]any)
	if !ok {
		shared.SaveError(pipelineID, "records_path is set but "+path+" is not a JSON object")
		return
	}
	nested, ok := obj[recordsPath]
	if !ok {
		shared.SaveError(pipelineID, "records_path \""+recordsPath+"\" not found in "+path)
		return
	}

	items, ok := nested.([]any)
	if !ok {
		items = []any{nested} // a single JSON object -> one record
	}

	for _, item := range items {
		if emitItem(ctx, pipelineID, path, item, out) {
			return // context was cancelled while sending
		}
	}
}

func streamJSONRecords(ctx context.Context, pipelineID, path string, r io.Reader, out chan<- shared.Record) error {
	decoder := json.NewDecoder(r)

	tok, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return errors.New("top-level JSON value must be an object or array")
	}

	switch delim {
	case '[':
		for decoder.More() {
			var item any
			if err := decoder.Decode(&item); err != nil {
				return err
			}
			if emitItem(ctx, pipelineID, path, item, out) {
				return nil // context was cancelled while sending
			}
		}
	case '{':
		fields := make(map[string]any)
		for decoder.More() {
			keyTok, err := decoder.Token()
			if err != nil {
				return err
			}
			var value any
			if err := decoder.Decode(&value); err != nil {
				return err
			}
			fields[keyTok.(string)] = value
		}
		sendRecord(ctx, fields, out)
	default:
		return errors.New("top-level JSON value must be an object or array")
	}
	return nil
}

func emitItem(ctx context.Context, pipelineID, path string, item any, out chan<- shared.Record) bool {
	fields, ok := item.(map[string]any)
	if !ok {
		shared.SaveError(pipelineID, "skipping non-object element in "+path)
		return false
	}
	return !sendRecord(ctx, fields, out)
}

func sendRecord(ctx context.Context, fields map[string]any, out chan<- shared.Record) bool {
	record := shared.Record{ID: uuid.NewString(), Fields: make(map[string]string, len(fields))}
	for key, value := range fields {
		record.Fields[key] = jsonValueToString(value)
	}

	select {
	case <-ctx.Done(): // pipeline was cancelled, stop reading immediately
		return false
	case out <- record:
		return true
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
