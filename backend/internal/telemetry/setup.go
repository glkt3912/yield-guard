package telemetry

import (
	"context"
	"errors"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// Package-level metric instruments. Initialised by Setup(); before that they are no-ops.
var (
	AnalyzeRequestsTotal    metric.Int64Counter
	MLITAPILatencyHistogram metric.Float64Histogram
	MLITCacheHits           metric.Int64Counter
	MLITCacheMisses         metric.Int64Counter
)

func init() {
	initInstruments(otel.GetMeterProvider())
}

func initInstruments(mp metric.MeterProvider) {
	meter := mp.Meter("yield-guard.backend")
	AnalyzeRequestsTotal, _ = meter.Int64Counter("analyze.requests.total",
		metric.WithDescription("Number of investment analysis requests"),
		metric.WithUnit("{request}"),
	)
	MLITAPILatencyHistogram, _ = meter.Float64Histogram("mlit.api.latency",
		metric.WithDescription("MLIT API request latency per attempt"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(50, 100, 250, 500, 1000, 2500, 5000, 30000),
	)
	MLITCacheHits, _ = meter.Int64Counter("mlit.cache.hits",
		metric.WithDescription("MLIT API cache hits"),
		metric.WithUnit("{hit}"),
	)
	MLITCacheMisses, _ = meter.Int64Counter("mlit.cache.misses",
		metric.WithDescription("MLIT API cache misses"),
		metric.WithUnit("{miss}"),
	)
}

// Setup initialises the global TracerProvider and MeterProvider.
// If OTEL_EXPORTER_OTLP_ENDPOINT is empty, stdout exporters are used (local dev).
// Returns a shutdown function that must be deferred by the caller.
func Setup(ctx context.Context, serviceName, serviceVersion string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// --- Trace exporter ---
	var traceExporter sdktrace.SpanExporter
	if otlpEndpoint == "" {
		traceExporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	} else {
		traceExporter, err = otlptracegrpc.New(ctx)
	}
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	)

	// --- Metric exporter ---
	var metricExporter sdkmetric.Exporter
	if otlpEndpoint == "" {
		metricExporter, err = stdoutmetric.New()
	} else {
		metricExporter, err = otlpmetricgrpc.New(ctx)
	}
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// Re-initialise instruments with the real provider.
	initInstruments(mp)

	shutdown := func(ctx context.Context) error {
		return errors.Join(mp.Shutdown(ctx), tp.Shutdown(ctx))
	}
	return shutdown, nil
}
