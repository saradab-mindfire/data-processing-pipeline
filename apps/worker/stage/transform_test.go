package stage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/shared"
)

func TestTransformTrimsFieldWhitespace(t *testing.T) {
	ctx := context.Background()
	in := make(chan shared.Record, 1)
	out := make(chan shared.Record, 1)

	in <- shared.Record{ID: "1", Fields: map[string]string{"name": "  Alice  ", "age": "30"}}
	close(in)

	var wg sync.WaitGroup
	wg.Add(1)
	go Transform(ctx, &wg, in, out)
	wg.Wait()
	close(out)

	record, ok := <-out
	if !ok {
		t.Fatal("expected a transformed record on the output channel")
	}
	if record.Fields["name"] != "Alice" {
		t.Errorf("name = %q, want %q", record.Fields["name"], "Alice")
	}
	if record.Fields["age"] != "30" {
		t.Errorf("age = %q, want %q", record.Fields["age"], "30")
	}
}

func TestTransformStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	in := make(chan shared.Record, 1)
	in <- shared.Record{ID: "1", Fields: map[string]string{"name": " Alice "}}
	out := make(chan shared.Record)

	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		Transform(ctx, &wg, in, out)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Transform did not return promptly after context cancellation")
	}
}
