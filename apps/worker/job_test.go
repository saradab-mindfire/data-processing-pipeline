package worker

import (
	"context"
	"sync"
	"testing"

	"github.com/saradab-mindfire/data-processing-pipeline/packages/queue"
)

func TestJobAddValidAddInvalidCounts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	j := &job{pipelineID: "job-counts-pipeline", ctx: ctx, cancel: cancel}

	j.addValid()
	j.addValid()
	j.addInvalid()

	processed, valid, invalid := j.counts()
	if processed != 3 {
		t.Errorf("processed = %d, want 3", processed)
	}
	if valid != 2 {
		t.Errorf("valid = %d, want 2", valid)
	}
	if invalid != 1 {
		t.Errorf("invalid = %d, want 1", invalid)
	}

	// addValid/addInvalid should also publish progress to the shared queue.
	qProcessed, qValid, qInvalid, ok := queue.GetProgress(j.pipelineID)
	if !ok {
		t.Fatal("expected progress to be recorded in queue")
	}
	if qProcessed != 3 || qValid != 2 || qInvalid != 1 {
		t.Errorf("queue progress = (%d,%d,%d), want (3,2,1)", qProcessed, qValid, qInvalid)
	}

	queue.DeleteProgress(j.pipelineID)
}

func TestJobCountsConcurrent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	j := &job{pipelineID: "job-concurrent-pipeline", ctx: ctx, cancel: cancel}

	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(iterations * 2)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			j.addValid()
		}()
		go func() {
			defer wg.Done()
			j.addInvalid()
		}()
	}
	wg.Wait()

	processed, valid, invalid := j.counts()
	if processed != iterations*2 {
		t.Errorf("processed = %d, want %d", processed, iterations*2)
	}
	if valid != iterations {
		t.Errorf("valid = %d, want %d", valid, iterations)
	}
	if invalid != iterations {
		t.Errorf("invalid = %d, want %d", invalid, iterations)
	}

	queue.DeleteProgress(j.pipelineID)
}

func TestRemoveJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const pipelineID = "job-remove-pipeline"
	jobsMu.Lock()
	jobs[pipelineID] = &job{pipelineID: pipelineID, ctx: ctx, cancel: cancel}
	jobsMu.Unlock()

	removeJob(pipelineID)

	jobsMu.Lock()
	_, ok := jobs[pipelineID]
	jobsMu.Unlock()
	if ok {
		t.Fatal("expected job to be removed from the jobs map")
	}

	// removeJob on an already-absent id must not panic.
	removeJob(pipelineID)
}
