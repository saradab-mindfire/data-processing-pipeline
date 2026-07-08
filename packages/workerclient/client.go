package workerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
)

var (
	baseURL       string
	exportBaseURL string
	httpClient    = &http.Client{Timeout: 10 * time.Second}
)

// Init sets the base URL used to reach the worker's internal API.
func Init(url string) {
	baseURL = url
}

func InitExportBaseURL(url string) {
	exportBaseURL = url
}

func ExportBaseURL() string {
	return exportBaseURL
}

// submits a pipeline job to the worker.
func Enqueue(ctx context.Context, pipelineID string, req models.PipelineRequest) error {
	body, err := json.Marshal(struct {
		PipelineID string                 `json:"pipeline_id"`
		Request    models.PipelineRequest `json:"request"`
	}{PipelineID: pipelineID, Request: req})
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/internal/pipelines", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("workerclient: enqueue failed with status %d", resp.StatusCode)
	}
	return nil
}

// Get progress of the pipeline
func GetProgress(pipelineID string) (processed, valid, invalid int, ok bool) {
	resp, err := httpClient.Get(baseURL + "/internal/pipelines/" + pipelineID + "/progress")
	if err != nil {
		return 0, 0, 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, false
	}

	var body struct {
		Processed int `json:"processed"`
		Valid     int `json:"valid"`
		Invalid   int `json:"invalid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, 0, 0, false
	}
	return body.Processed, body.Valid, body.Invalid, true
}

// Cancel the worker to cancel a running pipeline job
func Cancel(pipelineID string) error {
	resp, err := httpClient.Post(baseURL+"/internal/pipelines/"+pipelineID+"/cancel", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
