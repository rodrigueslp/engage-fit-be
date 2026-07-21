package observability

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type Config struct {
	Enabled          bool
	ServiceName      string
	ServiceVersion   string
	Environment      string
	TraceSampleRatio float64
	Prometheus       bool
}

type System struct {
	MetricsHandler http.Handler
	shutdown       []func(context.Context) error
}

func Setup(ctx context.Context, cfg Config) (*System, error) {
	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.DeploymentEnvironmentName(cfg.Environment),
	))
	if err != nil {
		return nil, err
	}

	system := &System{}
	meterOptions := []metric.Option{metric.WithResource(res)}
	if cfg.Prometheus {
		exporter, err := prometheus.New()
		if err != nil {
			return nil, err
		}
		meterOptions = append(meterOptions, metric.WithReader(exporter))
		system.MetricsHandler = promhttp.Handler()
	}
	if cfg.Enabled {
		metricExporter, err := otlpmetrichttp.New(ctx)
		if err != nil {
			return nil, err
		}
		meterOptions = append(meterOptions, metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(30*time.Second))))
	}
	meterProvider := metric.NewMeterProvider(meterOptions...)
	otel.SetMeterProvider(meterProvider)
	meter := otel.Meter("engagefit/application")
	_, err = meter.Int64ObservableGauge("engagefit.application.info",
		otelmetric.WithDescription("Informacao e heartbeat da aplicacao"),
		otelmetric.WithInt64Callback(func(_ context.Context, observer otelmetric.Int64Observer) error {
			observer.Observe(1, otelmetric.WithAttributes(attribute.String("service.version", cfg.ServiceVersion), attribute.String("deployment.environment", cfg.Environment)))
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	system.shutdown = append(system.shutdown, meterProvider.Shutdown)
	if err := otelruntime.Start(otelruntime.WithMeterProvider(meterProvider)); err != nil {
		return nil, err
	}

	stdoutHandler := slog.NewJSONHandler(os.Stdout, nil)
	if cfg.Enabled {
		traceExporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, err
		}
		tracerProvider := trace.NewTracerProvider(
			trace.WithResource(res),
			trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(cfg.TraceSampleRatio))),
			trace.WithBatcher(traceExporter),
		)
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
		system.shutdown = append(system.shutdown, tracerProvider.Shutdown)

		logExporter, err := otlploghttp.New(ctx)
		if err != nil {
			return nil, err
		}
		loggerProvider := log.NewLoggerProvider(log.WithResource(res), log.WithProcessor(log.NewBatchProcessor(logExporter)))
		otelHandler := otelslog.NewHandler(cfg.ServiceName, otelslog.WithLoggerProvider(loggerProvider))
		slog.SetDefault(slog.New(newMultiHandler(stdoutHandler, otelHandler)))
		system.shutdown = append(system.shutdown, loggerProvider.Shutdown)
	} else {
		slog.SetDefault(slog.New(stdoutHandler))
	}

	return system, nil
}

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return multiHandler{handlers: handlers}
}

func (h multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var result error
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, record.Level) {
			result = errors.Join(result, handler.Handle(ctx, record.Clone()))
		}
	}
	return result
}

func (h multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithAttrs(attrs))
	}
	return multiHandler{handlers: handlers}
}

func (h multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithGroup(name))
	}
	return multiHandler{handlers: handlers}
}

func (s *System) Shutdown(ctx context.Context) error {
	var result error
	for index := len(s.shutdown) - 1; index >= 0; index-- {
		result = errors.Join(result, s.shutdown[index](ctx))
	}
	return result
}
