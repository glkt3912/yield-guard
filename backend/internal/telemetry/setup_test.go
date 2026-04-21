package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestSetup_StdoutExporter(t *testing.T) {
	// Ensure OTEL_EXPORTER_OTLP_ENDPOINT is unset so stdout exporter is used.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	ctx := context.Background()
	shutdown, err := Setup(ctx, "test-service", "0.0.1")
	if err != nil {
		t.Fatalf("Setup() returned unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Setup() returned nil shutdown function")
	}

	// Global providers should be the real SDK providers after Setup.
	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Error("global TracerProvider is not *sdktrace.TracerProvider after Setup")
	}
	if _, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider); !ok {
		t.Error("global MeterProvider is not *sdkmetric.MeterProvider after Setup")
	}

	// First shutdown should succeed.
	if err := shutdown(ctx); err != nil {
		t.Errorf("first shutdown returned error: %v", err)
	}

	// Second shutdown should not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("second shutdown panicked: %v", r)
		}
	}()
	_ = shutdown(ctx)
}

func TestSetup_InstrumentsInitialised(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	ctx := context.Background()
	shutdown, err := Setup(ctx, "test-service", "0.0.1")
	if err != nil {
		t.Fatalf("Setup() error: %v", err)
	}
	defer func() { _ = shutdown(ctx) }()

	// Instruments should be callable without panic after Setup.
	AnalyzeRequestsTotal.Add(ctx, 1)
	MLITAPILatencyHistogram.Record(ctx, 42.5)
	MLITCacheHits.Add(ctx, 1)
	MLITCacheMisses.Add(ctx, 1)
}
