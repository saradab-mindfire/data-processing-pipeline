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
