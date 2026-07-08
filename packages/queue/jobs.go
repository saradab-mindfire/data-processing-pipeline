package queue

import (
	"context"

	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
)

type Job struct {
	PipelineID string                 `json:"pipeline_id"`
	Request    models.PipelineRequest `json:"request"`
}

// jobs is the in-memory pipeline 
var jobs = make(chan Job, 100)

// Enqueue queues a pipeline job for the worker to pick up.
func Enqueue(ctx context.Context, pipelineID string, req models.PipelineRequest) error {
	select {
	case jobs <- Job{PipelineID: pipelineID, Request: req}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Dequeue blocks until a pipeline job is available and returns it.
func Dequeue(ctx context.Context) (Job, error) {
	select {
	case job := <-jobs:
		return job, nil
	case <-ctx.Done():
		return Job{}, ctx.Err()
	}
}
