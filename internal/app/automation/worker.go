package automation

import (
	"context"
	"log"
	"time"
)

type Worker struct {
	execute  ExecuteScheduleUseCase
	interval time.Duration
}

func NewWorker(execute ExecuteScheduleUseCase, interval time.Duration) Worker {
	if interval <= 0 {
		interval = time.Minute
	}
	return Worker{execute: execute, interval: interval}
}

func (w Worker) Start(ctx context.Context) {
	go func() {
		w.tick(ctx)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.tick(ctx)
			}
		}
	}()
}

func (w Worker) tick(ctx context.Context) {
	runs, err := w.execute.ExecuteDue(ctx, time.Now())
	if err != nil {
		log.Printf("automation worker failed: %v", err)
		return
	}
	for _, run := range runs {
		log.Printf("automation schedule run finished: id=%s status=%s sent=%d failed=%d", run.ID, run.Status, run.SentMessages, run.FailedMessages)
	}
}
