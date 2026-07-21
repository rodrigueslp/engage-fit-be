package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func RecordImport(ctx context.Context, source, status string, records, checkins int, elapsed time.Duration) {
	meter := otel.Meter("engagefit/imports")
	operations, _ := meter.Int64Counter("engagefit.imports.operations", metric.WithDescription("Importacoes finalizadas"))
	rows, _ := meter.Int64Counter("engagefit.imports.records", metric.WithDescription("Linhas lidas em importacoes"))
	inserted, _ := meter.Int64Counter("engagefit.imports.checkins", metric.WithDescription("Check-ins inseridos"))
	duration, _ := meter.Float64Histogram("engagefit.imports.duration", metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(0.1, 0.5, 1, 2.5, 5, 10, 30, 60))
	attrs := metric.WithAttributes(attribute.String("import.source", source), attribute.String("import.status", status))
	operations.Add(ctx, 1, attrs)
	if records > 0 {
		rows.Add(ctx, int64(records), attrs)
	}
	if checkins > 0 {
		inserted.Add(ctx, int64(checkins), attrs)
	}
	duration.Record(ctx, elapsed.Seconds(), attrs)
}

func RecordGateway(ctx context.Context, gateway, operation, status string, elapsed time.Duration) {
	meter := otel.Meter("engagefit/gateways")
	calls, _ := meter.Int64Counter("engagefit.gateway.calls", metric.WithDescription("Chamadas a gateways externos"))
	duration, _ := meter.Float64Histogram("engagefit.gateway.duration", metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60))
	attrs := metric.WithAttributes(attribute.String("gateway.name", gateway), attribute.String("gateway.operation", operation), attribute.String("gateway.status", status))
	calls.Add(ctx, 1, attrs)
	duration.Record(ctx, elapsed.Seconds(), attrs)
}

func RecordStaleAutomationRuns(ctx context.Context, count int64) {
	if count <= 0 {
		return
	}
	meter := otel.Meter("engagefit/automation")
	counter, _ := meter.Int64Counter("engagefit.automation.stale_runs", metric.WithDescription("Automacoes marcadas como stale"))
	counter.Add(ctx, count)
}
