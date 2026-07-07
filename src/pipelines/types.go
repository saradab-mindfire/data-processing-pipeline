package pipelines

type SourceConfig struct {
	Type string `json:"type"` // "csv" or "json"
	Path string `json:"path"` // local file path, or an http(s):// URL

	RecordsPath string `json:"records_path,omitempty"`
}

const (
	validationWorkers = 5
	transformWorkers  = 3
)

type PipelineRequest struct {
	Sources    []SourceConfig `json:"sources"`
	ExportType string         `json:"export_type"`
}

type Record struct {
	ID     string
	Fields map[string]string
}
