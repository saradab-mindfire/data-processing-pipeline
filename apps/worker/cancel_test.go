package worker

import (
	"context"
	"testing"
)

func TestCancelKnownJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	const pipelineID = "cancel-known-pipeline"

	jobsMu.Lock()
	jobs[pipelineID] = &job{pipelineID: pipelineID, ctx: ctx, cancel: cancel}
	jobsMu.Unlock()
	t.Cleanup(func() { removeJob(pipelineID) })

	ok := Cancel(pipelineID)
	if !ok {
		t.Fatal("Cancel should return true for a known pipeline")
	}

	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected job context to be cancelled")
	}
}

func TestCancelUnknownJob(t *testing.T) {
	ok := Cancel("cancel-unknown-pipeline")
	if ok {
		t.Fatal("Cancel should return false for an unknown pipeline")
	}
}
