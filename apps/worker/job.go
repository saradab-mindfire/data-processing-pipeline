package worker

import (
	"context"
	"sync"

	"github.com/saradab-mindfire/data-processing-pipeline/packages/queue"
)

type job struct {
	pipelineID string
	ctx        context.Context
	cancel     context.CancelFunc

	mu        sync.Mutex // several workers touch the counters below at once
	processed int
	valid     int
	invalid   int
}

func (j *job) addValid() {
	j.mu.Lock()
	j.processed++
	j.valid++
	processed, valid, invalid := j.processed, j.valid, j.invalid
	j.mu.Unlock()
	queue.SetProgress(j.pipelineID, processed, valid, invalid)
}

func (j *job) addInvalid() {
	j.mu.Lock()
	j.processed++
	j.invalid++
	processed, valid, invalid := j.processed, j.valid, j.invalid
	j.mu.Unlock()
	queue.SetProgress(j.pipelineID, processed, valid, invalid)
}

func (j *job) counts() (processed, valid, invalid int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.processed, j.valid, j.invalid
}

var (
	jobsMu sync.Mutex
	jobs   = map[string]*job{}
)

func removeJob(pipelineID string) {
	jobsMu.Lock()
	delete(jobs, pipelineID)
	jobsMu.Unlock()
}
