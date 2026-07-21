package automation

import (
	"context"
	"log/slog"
	"time"

	"boxengage/backend/internal/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Worker struct {
	execute  DueScheduleExecutor
	interval time.Duration
}

type DueScheduleExecutor interface {
	ExecuteDue(ctx context.Context, now time.Time) ([]domain.AutomationRun, error)
}

func NewWorker(execute DueScheduleExecutor, interval time.Duration) Worker {
	if interval <= 0 {
		interval = time.Minute
	}
	return Worker{execute: execute, interval: interval}
}

func (w Worker) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
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
	return done
}

func (w Worker) tick(ctx context.Context) {
	runs, err := w.execute.ExecuteDue(ctx, time.Now())
	if err != nil {
		slog.ErrorContext(ctx, "automation_worker_tick_failed", "error", err)
	}
	meter := otel.Meter("engagefit/automation")
	runCounter, _ := meter.Int64Counter("engagefit.automation.runs", metric.WithDescription("Execucoes de automacao concluidas"))
	runDuration, _ := meter.Float64Histogram("engagefit.automation.duration", metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(1, 5, 15, 30, 60, 120, 300, 600))
	for _, run := range runs {
		attrs := metric.WithAttributes(attribute.String("automation.status", run.Status), attribute.String("automation.mode", run.Mode))
		runCounter.Add(ctx, 1, attrs)
		if run.FinishedAt != nil {
			runDuration.Record(ctx, run.FinishedAt.Sub(run.StartedAt).Seconds(), attrs)
		}
		slog.InfoContext(ctx, "automation_schedule_finished", "run_id", run.ID, "schedule_id", run.ScheduleID, "status", run.Status, "sent_messages", run.SentMessages, "failed_messages", run.FailedMessages)
	}
}
