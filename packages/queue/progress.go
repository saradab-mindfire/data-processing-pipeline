package queue

import "sync"

type progressEntry struct {
	processed int
	valid     int
	invalid   int
}

var (
	progressMu sync.RWMutex
	progress   = map[string]progressEntry{}
)

// SetProgress records for a running pipeline.
func SetProgress(pipelineID string, processed, valid, invalid int) {
	progressMu.Lock()
	defer progressMu.Unlock()
	progress[pipelineID] = progressEntry{processed: processed, valid: valid, invalid: invalid}
}

// GetProgress of a running pipeline
func GetProgress(pipelineID string) (processed, valid, invalid int, ok bool) {
	progressMu.RLock()
	defer progressMu.RUnlock()

	entry, ok := progress[pipelineID]
	if !ok {
		return 0, 0, 0, false
	}
	return entry.processed, entry.valid, entry.invalid, true
}

// Delete progress using pipeline id
func DeleteProgress(pipelineID string) {
	progressMu.Lock()
	defer progressMu.Unlock()
	delete(progress, pipelineID)
}
