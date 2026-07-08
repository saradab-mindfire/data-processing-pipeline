package dataio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

func TestExportResultWritesJSONFile(t *testing.T) {
	pipelineID := "export-pipeline"
	exportPath := filepath.Join("exports", pipelineID+".json")
	t.Cleanup(func() { os.Remove(exportPath) })

	records := []shared.Record{
		{ID: "1", Fields: map[string]string{"name": "Alice"}},
		{ID: "2", Fields: map[string]string{"name": "Bob"}},
	}

	ExportResult(pipelineID, records)

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("expected export file to exist: %v", err)
	}

	var parsed struct {
		TotalTransformed int             `json:"total_transformed"`
		Records          []shared.Record `json:"records"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse export file: %v", err)
	}
	if parsed.TotalTransformed != 2 {
		t.Errorf("total_transformed = %d, want 2", parsed.TotalTransformed)
	}
	if len(parsed.Records) != 2 {
		t.Errorf("got %d records, want 2", len(parsed.Records))
	}
}

func TestExportResultEmptyRecords(t *testing.T) {
	pipelineID := "export-empty-pipeline"
	exportPath := filepath.Join("exports", pipelineID+".json")
	t.Cleanup(func() { os.Remove(exportPath) })

	ExportResult(pipelineID, []shared.Record{})

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("expected export file to exist: %v", err)
	}

	var parsed struct {
		TotalTransformed int `json:"total_transformed"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse export file: %v", err)
	}
	if parsed.TotalTransformed != 0 {
		t.Errorf("total_transformed = %d, want 0", parsed.TotalTransformed)
	}
}
