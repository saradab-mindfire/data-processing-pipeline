package dataio

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

func writeUploadFixture(t *testing.T, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(localSourceRoot, 0o755); err != nil {
		t.Fatalf("failed to create uploads dir: %v", err)
	}
	path := filepath.Join(localSourceRoot, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })
	return name
}

func TestReadCSVParsesRows(t *testing.T) {
	name := writeUploadFixture(t, "readcsv_basic.csv", "name,age\nAlice,30\nBob,25\n")

	out := make(chan shared.Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadCSV(context.Background(), &wg, "csv-pipeline", name, out)
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
	if records[1].Fields["name"] != "Bob" || records[1].Fields["age"] != "25" {
		t.Errorf("unexpected second record: %+v", records[1])
	}
	if records[0].ID == "" || records[0].ID == records[1].ID {
		t.Errorf("expected unique, non-empty record IDs, got %q and %q", records[0].ID, records[1].ID)
	}
}

func TestReadCSVMissingFileRecordsError(t *testing.T) {
	out := make(chan shared.Record, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	ReadCSV(context.Background(), &wg, "csv-missing-pipeline", "does-not-exist.csv", out)
	wg.Wait()
	close(out)

	if _, ok := <-out; ok {
		t.Fatal("expected no records to be emitted for a missing file")
	}
}

func TestReadCSVStopsOnContextCancel(t *testing.T) {
	name := writeUploadFixture(t, "readcsv_cancel.csv", "name\nAlice\nBob\nCarol\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out := make(chan shared.Record) // unbuffered so a blocked send would hang forever
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		ReadCSV(ctx, &wg, "csv-cancel-pipeline", name, out)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReadCSV did not return promptly after context cancellation")
	}
}
