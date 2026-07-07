package worker

func Progress(pipelineID string) (processed, valid, invalid int, ok bool) {
	jobsMu.Lock()
	j, found := jobs[pipelineID]
	jobsMu.Unlock()
	if !found {
		return 0, 0, 0, false
	}
	processed, valid, invalid = j.counts()
	return processed, valid, invalid, true
}