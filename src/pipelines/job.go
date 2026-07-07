package pipelines

import (
	"context"
	"sync"
)

type job struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu        sync.Mutex // several workers touch the counters below at once
	processed int
	valid     int
	invalid   int
}

func (j *job) addValid() {
	j.mu.Lock()
	j.processed++
	j.valid++
	j.mu.Unlock()
}

func (j *job) addInvalid() {
	j.mu.Lock()
	j.processed++
	j.invalid++
	j.mu.Unlock()
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
