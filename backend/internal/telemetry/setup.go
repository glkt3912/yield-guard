package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	monitoring "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// gcpAuthTransport injects a GCP OAuth2 Bearer token into every outbound request.
type gcpAuthTransport struct {
	base     http.RoundTripper
	tokenSrc oauth2.TokenSource
}

func (t *gcpAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.tokenSrc.Token()
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return t.base.RoundTrip(req)
}

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
	MLITAPILatencyHistogram, _ = meter.Float64Histogram("mlit.api.request.duration",
		metric.WithDescription("MLIT API request duration per attempt"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 30),
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
// If GOOGLE_CLOUD_PROJECT is set, Google Cloud Trace/Monitoring exporters are used (production).
// Otherwise stdout exporters are used (local dev).
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

	gcpProject := os.Getenv("GOOGLE_CLOUD_PROJECT")

	// --- Trace exporter ---
	var traceExporter sdktrace.SpanExporter
	if gcpProject == "" {
		traceExporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	} else {
		otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
		if otlpEndpoint == "" {
			otlpEndpoint = "https://telemetry.googleapis.com/v1/traces"
		}
		var tokenSrc oauth2.TokenSource
		tokenSrc, err = google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/trace.append")
		if err != nil {
			return nil, fmt.Errorf("GCP trace auth: %w", err)
		}
		traceExporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(otlpEndpoint),
			otlptracehttp.WithHTTPClient(&http.Client{
				Transport: &gcpAuthTransport{base: http.DefaultTransport, tokenSrc: tokenSrc},
			}),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
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
	if gcpProject == "" {
		metricExporter, err = stdoutmetric.New()
	} else {
		metricExporter, err = monitoring.New()
	}
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// Re-initialise instruments with the real provider.
	initInstruments(mp)

	shutdown := func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
	}
	return shutdown, nil
}
