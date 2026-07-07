package pipelines

// Cancel stops a running pipeline early. Returns false if it isn't running
// in memory anymore (already finished, or the server restarted since it started).
// Flow: called by controllers.CancelPipeline.
func Cancel(pipelineID string) bool {
	jobsMu.Lock()
	j, ok := jobs[pipelineID]
	jobsMu.Unlock()
	if !ok {
		return false
	}
	j.cancel() // every worker's `select { case <-ctx.Done(): ... }` below reacts to this
	return true
}