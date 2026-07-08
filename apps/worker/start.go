package worker

import (
	"context"

	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
)

func Start(pipelineID string, req models.PipelineRequest) {
	ctx, cancel := context.WithCancel(context.Background())
	j := &job{pipelineID: pipelineID, ctx: ctx, cancel: cancel}

	jobsMu.Lock()
	jobs[pipelineID] = j
	jobsMu.Unlock()

	go run(j, pipelineID, req)
}