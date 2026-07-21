package observability

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestBusinessMetricsUseBoundedDimensions(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)
	defer otel.SetMeterProvider(noop.NewMeterProvider())

	ctx := context.Background()
	RecordImport(ctx, "totalpass", "success", 10, 8, time.Second)
	RecordGateway(ctx, "smtp", "send", "error", 2*time.Second)
	RecordStaleAutomationRuns(ctx, 2)

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &metrics); err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{"engagefit.imports.operations": false, "engagefit.gateway.calls": false, "engagefit.automation.stale_runs": false}
	for _, scope := range metrics.ScopeMetrics {
		for _, current := range scope.Metrics {
			if _, ok := wanted[current.Name]; ok {
				wanted[current.Name] = true
			}
		}
	}
	for name, found := range wanted {
		if !found {
			t.Errorf("metric %s was not collected", name)
		}
	}
}
