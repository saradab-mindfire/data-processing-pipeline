package worker

// Cancel stops the job for pipelineID if it is running on this worker,
// returning whether a matching job was found.
func Cancel(pipelineID string) bool {
	jobsMu.Lock()
	j, ok := jobs[pipelineID]
	jobsMu.Unlock()
	if !ok {
		return false
	}
	j.cancel()
	return true
}
