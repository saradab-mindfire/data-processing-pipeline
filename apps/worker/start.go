package worker

import "context"

func Start(pipelineID string, req PipelineRequest) {
	ctx, cancel := context.WithCancel(context.Background())
	j := &job{ctx: ctx, cancel: cancel}

	jobsMu.Lock()
	jobs[pipelineID] = j
	jobsMu.Unlock()

	go run(j, pipelineID, req)
}