package worker

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