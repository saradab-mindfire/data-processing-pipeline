package dataio

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

// ExportResult writes the transformed records to exports/<pipeline-id>.json.
func ExportResult(pipelineID string, records []shared.Record) {
	dir := "exports"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		shared.SaveError(pipelineID, "could not create exports folder: "+err.Error())
		return
	}

	data, _ := json.MarshalIndent(map[string]any{
		"total_transformed": len(records),
		"records":           records,
	}, "", "  ")

	path := filepath.Join(dir, pipelineID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		shared.SaveError(pipelineID, "could not write export file: "+err.Error())
	}
}
