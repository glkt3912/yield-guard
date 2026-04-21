package mlit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/yield-guard/backend/internal/domain"
)

// setupTestTracer installs an in-memory exporter as the global TracerProvider
// for the duration of the test and returns the exporter for assertions.
func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	return exp
}

// newOtelTestClient creates a Client whose httpClient points at the given test server.
func newOtelTestClient(srv *httptest.Server) *Client {
	return &Client{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
		apiKey:     "test-key",
		cache:      newCache(),
	}
}

// makeSuccessHandler returns an httptest handler that responds with a minimal
// valid XIT001 JSON payload containing one land transaction.
func makeSuccessHandler() http.HandlerFunc {
	tx := Transaction{
		Type:         "宅地(土地)",
		TradePrice:   "10000000",
		Area:         "100",
		PricePerUnit: "100000",
		Period:       "2024Q1",
		DistrictName: "Test District",
	}
	resp := APIResponse{
		Status: "OK",
		Data:   []Transaction{tx},
	}
	body, _ := json.Marshal(resp)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

func TestFetchLandPrices_SpanCreated(t *testing.T) {
	exp := setupTestTracer(t)

	srv := httptest.NewServer(makeSuccessHandler())
	defer srv.Close()

	client := newOtelTestClient(srv)
	q := LandPriceQuery{Area: "13", Year: 2024, Quarter: 1, ToYear: 2024, ToQuarter: 4}

	_, err := client.FetchLandPrices(context.Background(), q)
	if err != nil {
		t.Fatalf("FetchLandPrices returned unexpected error: %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span, got none")
	}

	var found bool
	for _, s := range spans {
		if s.Name == "mlit.FetchLandPrices" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("span named 'mlit.FetchLandPrices' not found; got: %v", spanNames(spans))
	}
}

func TestFetchLandPrices_CacheHitAttribute(t *testing.T) {
	exp := setupTestTracer(t)

	srv := httptest.NewServer(makeSuccessHandler())
	defer srv.Close()

	client := newOtelTestClient(srv)
	q := LandPriceQuery{Area: "13", Year: 2024, Quarter: 1, ToYear: 2024, ToQuarter: 4}

	// First call — cache miss.
	_, err := client.FetchLandPrices(context.Background(), q)
	if err != nil {
		t.Fatalf("first FetchLandPrices error: %v", err)
	}

	exp.Reset()

	// Second call — cache hit.
	_, err = client.FetchLandPrices(context.Background(), q)
	if err != nil {
		t.Fatalf("second FetchLandPrices error: %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans on second call")
	}

	var cacheHit *bool
	for _, s := range spans {
		if s.Name != "mlit.FetchLandPrices" {
			continue
		}
		for _, attr := range s.Attributes {
			if string(attr.Key) == "mlit.cache.hit" {
				v := attr.Value.AsBool()
				cacheHit = &v
			}
		}
	}
	if cacheHit == nil {
		t.Fatal("attribute 'mlit.cache.hit' not found on span")
	}
	if !*cacheHit {
		t.Errorf("expected mlit.cache.hit=true on second call, got false")
	}
}

func TestFetchLandPrices_ErrorSpanStatus(t *testing.T) {
	exp := setupTestTracer(t)

	// Server always returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newOtelTestClient(srv)
	// Override retry settings to make the test fast.
	q := LandPriceQuery{Area: "13", Year: 2024, Quarter: 1, ToYear: 2024, ToQuarter: 4}

	_, err := client.FetchLandPrices(context.Background(), q)
	if err == nil {
		t.Fatal("expected error from 500 server, got nil")
	}

	spans := exp.GetSpans()
	var foundError bool
	for _, s := range spans {
		if s.Name == "mlit.FetchLandPrices" && s.Status.Code.String() == "Error" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("expected span with Error status for HTTP 500 response; spans: %v", spanNames(spans))
	}
}

// spanNames returns a slice of span names for diagnostic output.
func spanNames(spans tracetest.SpanStubs) []string {
	names := make([]string, len(spans))
	for i, s := range spans {
		names[i] = s.Name
	}
	return names
}

// Ensure domain import is used (referenced by makeSuccessHandler indirectly through parseTransactions).
var _ = domain.LandTransaction{}
