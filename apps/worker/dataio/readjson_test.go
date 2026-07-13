package dataio

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

func TestReadJSONArrayOfObjects(t *testing.T) {
	name := writeUploadFixture(t, "readjson_array.json", `[{"name":"Alice","age":30},{"name":"Bob","age":25}]`)

	out := make(chan shared.Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadJSON(context.Background(), &wg, "json-pipeline", name, "", out)
	wg.Wait()
	close(out)

	var records []shared.Record
	for record := range out {
		records = append(records, record)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0].Fields["name"] != "Alice" || records[0].Fields["age"] != "30" {
		t.Errorf("unexpected first record: %+v", records[0])
	}
}

func TestReadJSONSingleObject(t *testing.T) {
	name := writeUploadFixture(t, "readjson_single.json", `{"name":"Alice","age":30}`)

	out := make(chan shared.Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadJSON(context.Background(), &wg, "json-single-pipeline", name, "", out)
	wg.Wait()
	close(out)

	var records []shared.Record
	for record := range out {
		records = append(records, record)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Fields["name"] != "Alice" {
		t.Errorf("unexpected record: %+v", records[0])
	}
}

func TestReadJSONRecordsPath(t *testing.T) {
	name := writeUploadFixture(t, "readjson_records_path.json", `{"meta":{"count":2},"items":[{"name":"Alice"},{"name":"Bob"}]}`)

	out := make(chan shared.Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadJSON(context.Background(), &wg, "json-recordspath-pipeline", name, "items", out)
	wg.Wait()
	close(out)

	var records []shared.Record
	for record := range out {
		records = append(records, record)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
}

func TestReadJSONRecordsPathNotFound(t *testing.T) {
	name := writeUploadFixture(t, "readjson_records_path_missing.json", `{"meta":{"count":2}}`)

	out := make(chan shared.Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadJSON(context.Background(), &wg, "json-recordspath-missing-pipeline", name, "items", out)
	wg.Wait()
	close(out)

	if _, ok := <-out; ok {
		t.Fatal("expected no records when records_path is missing")
	}
}

func TestReadJSONSkipsNonObjectElements(t *testing.T) {
	name := writeUploadFixture(t, "readjson_mixed.json", `[{"name":"Alice"}, "not-an-object", {"name":"Bob"}]`)

	out := make(chan shared.Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadJSON(context.Background(), &wg, "json-mixed-pipeline", name, "", out)
	wg.Wait()
	close(out)

	var records []shared.Record
	for record := range out {
		records = append(records, record)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2 (non-object element should be skipped)", len(records))
	}
}

func TestReadJSONStreamsTopLevelObjectAsSingleRecord(t *testing.T) {
	name := writeUploadFixture(t, "readjson_stream_object.json", `{"name":"Alice","age":30}`)

	out := make(chan shared.Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadJSON(context.Background(), &wg, "json-stream-object-pipeline", name, "", out)
	wg.Wait()
	close(out)

	var records []shared.Record
	for record := range out {
		records = append(records, record)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Fields["name"] != "Alice" || records[0].Fields["age"] != "30" {
		t.Errorf("unexpected record: %+v", records[0])
	}
}

func TestReadJSONStreamRejectsNonObjectTopLevelValue(t *testing.T) {
	name := writeUploadFixture(t, "readjson_stream_scalar.json", `"just a string"`)

	out := make(chan shared.Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadJSON(context.Background(), &wg, "json-stream-scalar-pipeline", name, "", out)
	wg.Wait()
	close(out)

	if _, ok := <-out; ok {
		t.Fatal("expected no records for a top-level scalar JSON value")
	}
}

func TestReadJSONStreamRejectsMalformedJSON(t *testing.T) {
	name := writeUploadFixture(t, "readjson_stream_malformed.json", `{"name": "Alice",`)

	out := make(chan shared.Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadJSON(context.Background(), &wg, "json-stream-malformed-pipeline", name, "", out)
	wg.Wait()
	close(out)

	if _, ok := <-out; ok {
		t.Fatal("expected no records for malformed JSON")
	}
}

func TestReadJSONStreamStopsOnContextCancel(t *testing.T) {
	name := writeUploadFixture(t, "readjson_stream_cancel.json", `[{"name":"Alice"},{"name":"Bob"},{"name":"Carol"}]`)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out := make(chan shared.Record) // unbuffered, so a send blocks unless something reads
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		ReadJSON(ctx, &wg, "json-stream-cancel-pipeline", name, "", out)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReadJSON did not return promptly after context cancellation")
	}
	wg.Wait()
}

func TestJSONValueToString(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"number", float64(42), "42"},
		{"bool", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonValueToString(tt.value)
			if got != tt.want {
				t.Errorf("jsonValueToString(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
