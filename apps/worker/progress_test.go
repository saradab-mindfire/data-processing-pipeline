package worker

import (
	"context"
	"testing"
)

func TestProgressKnownJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const pipelineID = "progress-known-pipeline"

	j := &job{pipelineID: pipelineID, ctx: ctx, cancel: cancel}
	j.addValid()
	j.addInvalid()

	jobsMu.Lock()
	jobs[pipelineID] = j
	jobsMu.Unlock()
	t.Cleanup(func() { removeJob(pipelineID) })

	processed, valid, invalid, ok := Progress(pipelineID)
	if !ok {
		t.Fatal("expected Progress to find the job")
	}
	if processed != 2 || valid != 1 || invalid != 1 {
		t.Errorf("Progress = (%d,%d,%d), want (2,1,1)", processed, valid, invalid)
	}
}

func TestProgressUnknownJob(t *testing.T) {
	processed, valid, invalid, ok := Progress("progress-unknown-pipeline")
	if ok {
		t.Fatal("expected Progress to report ok=false for an unknown pipeline")
	}
	if processed != 0 || valid != 0 || invalid != 0 {
		t.Errorf("Progress = (%d,%d,%d), want zero values", processed, valid, invalid)
	}
}
