package models

type SourceConfig struct {
	Type string `json:"type"` // "csv" or "json"
	Path string `json:"path"` // local file path, or an http(s):// URL

	RecordsPath string `json:"records_path,omitempty"`
}

type PipelineRequest struct {
	Sources    []SourceConfig `json:"sources"`
	ExportType string         `json:"export_type"`
}
