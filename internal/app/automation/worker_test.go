package automation

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

type workerExecutor struct{ calls atomic.Int32 }

func (e *workerExecutor) ExecuteDue(context.Context, time.Time) ([]domain.AutomationRun, error) {
	e.calls.Add(1)
	return nil, nil
}

func TestWorkerStopsWhenContextIsCancelled(t *testing.T) {
	executor := &workerExecutor{}
	ctx, cancel := context.WithCancel(context.Background())
	done := NewWorker(executor, time.Hour).Start(ctx)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after cancellation")
	}
	if executor.calls.Load() > 1 {
		t.Fatalf("expected at most one initial tick, got %d", executor.calls.Load())
	}
}
