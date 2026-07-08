package shared

import (
	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
)

func SaveError(pipelineID, message string) {
	if database.Instance == nil {
		return
	}
	database.Instance.Create(&models.PipelineError{
		PipelineID: pipelineID,
		Message:    message,
	})
}

func SaveErrors(pipelineID string, messages []string) {
	if database.Instance == nil || len(messages) == 0 {
		return
	}
	errs := make([]models.PipelineError, len(messages))
	for i, message := range messages {
		errs[i] = models.PipelineError{PipelineID: pipelineID, Message: message}
	}
	database.Instance.Create(&errs)
}
