package stage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

func TestValidateSeparatesValidAndInvalidRecords(t *testing.T) {
	ctx := context.Background()
	in := make(chan shared.Record, 3)
	out := make(chan shared.Record, 3)

	in <- shared.Record{ID: "1", Fields: map[string]string{"name": "Alice"}}
	in <- shared.Record{ID: "2", Fields: map[string]string{"name": "", "age": "  "}}
	in <- shared.Record{ID: "3", Fields: map[string]string{}}
	close(in)

	var validCount, invalidCount int
	var mu sync.Mutex
	addValid := func() { mu.Lock(); validCount++; mu.Unlock() }
	addInvalid := func() { mu.Lock(); invalidCount++; mu.Unlock() }

	var wg sync.WaitGroup
	wg.Add(1)
	go Validate(ctx, &wg, "validate-pipeline", addValid, addInvalid, in, out)

	wg.Wait()
	close(out)

	if validCount != 1 {
		t.Errorf("validCount = %d, want 1", validCount)
	}
	if invalidCount != 2 {
		t.Errorf("invalidCount = %d, want 2", invalidCount)
	}

	var forwarded []shared.Record
	for record := range out {
		forwarded = append(forwarded, record)
	}
	if len(forwarded) != 1 || forwarded[0].ID != "1" {
		t.Errorf("expected only the valid record to be forwarded, got %+v", forwarded)
	}
}

func TestValidateStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	in := make(chan shared.Record, 1)
	in <- shared.Record{ID: "1", Fields: map[string]string{"name": "Alice"}}
	out := make(chan shared.Record)

	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		Validate(ctx, &wg, "validate-cancel-pipeline", func() {}, func() {}, in, out)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Validate did not return promptly after context cancellation")
	}
}

func TestHasNonEmptyField(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]string
		want   bool
	}{
		{"empty map", map[string]string{}, false},
		{"all blank", map[string]string{"a": "", "b": "   "}, false},
		{"one populated", map[string]string{"a": "", "b": "x"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasNonEmptyField(shared.Record{Fields: tt.fields})
			if got != tt.want {
				t.Errorf("hasNonEmptyField(%v) = %v, want %v", tt.fields, got, tt.want)
			}
		})
	}
}
