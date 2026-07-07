package pipelines

import (
	"github.com/saradab-mindfire/data-processing-pipeline/src/database"
	"github.com/saradab-mindfire/data-processing-pipeline/src/models"
)

// saveError records one problem as a PipelineError row, which is what
// GET /:id/errors reads back. Called directly wherever something goes wrong
// above - no separate "error collector" goroutine needed.
func saveError(pipelineID, message string) {
	if database.Instance == nil { // no DB connected (e.g. running tests) - skip
		return
	}
	database.Instance.Create(&models.PipelineError{
		PipelineID: pipelineID,
		Message:    message,
	})
}