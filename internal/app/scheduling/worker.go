package scheduling

import (
	"context"
	"log/slog"
	"time"
)

type MaterializerWorker struct {
	service  *GridService
	interval time.Duration
}

func NewMaterializerWorker(service *GridService, interval time.Duration) *MaterializerWorker {
	if interval < time.Minute {
		interval = time.Hour
	}
	return &MaterializerWorker{service: service, interval: interval}
}

func (w *MaterializerWorker) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		w.run(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.run(ctx)
			}
		}
	}()
	return done
}

func (w *MaterializerWorker) run(ctx context.Context) {
	created, err := w.service.Materialize(ctx)
	if err != nil {
		slog.Error("scheduling_materializer_failed", "error", err)
		return
	}
	slog.Info("scheduling_materializer_completed", "sessions_created", created)
}
