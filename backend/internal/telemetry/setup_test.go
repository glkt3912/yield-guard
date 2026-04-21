package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
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

	// Global providers should be registered after Setup.
	tp := otel.GetTracerProvider()
	if tp == nil {
		t.Error("TracerProvider is nil after Setup")
	}
	mp := otel.GetMeterProvider()
	if mp == nil {
		t.Error("MeterProvider is nil after Setup")
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

	if AnalyzeRequestsTotal == nil {
		t.Error("AnalyzeRequestsTotal is nil after Setup")
	}
	if MLITAPILatencyHistogram == nil {
		t.Error("MLITAPILatencyHistogram is nil after Setup")
	}
	if MLITCacheHits == nil {
		t.Error("MLITCacheHits is nil after Setup")
	}
	if MLITCacheMisses == nil {
		t.Error("MLITCacheMisses is nil after Setup")
	}

	// Instruments should be callable without panic.
	AnalyzeRequestsTotal.Add(ctx, 1)
	MLITAPILatencyHistogram.Record(ctx, 42.5)
	MLITCacheHits.Add(ctx, 1)
	MLITCacheMisses.Add(ctx, 1)
}
