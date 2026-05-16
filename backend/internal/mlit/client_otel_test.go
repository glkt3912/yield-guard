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

// makeGeoJSONHandler returns an httptest handler that responds with the given GeoJSON body.
func makeGeoJSONHandler(body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

// ---- FetchLocationOptimization (XKT003) ----

func TestFetchLocationOptimization_SpanCreated(t *testing.T) {
	exp := setupTestTracer(t)

	resp := LocationOptimizationGeoJSON{
		Type: "FeatureCollection",
		Features: []LocationOptimizationFeature{
			{Properties: LocationOptimizationProperties{KubunNameJa: "居住誘導区域"}},
		},
	}
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(makeGeoJSONHandler(body))
	defer srv.Close()

	client := newOtelTestClient(srv)
	_, err := client.FetchLocationOptimization(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("FetchLocationOptimization returned unexpected error: %v", err)
	}

	spans := exp.GetSpans()
	var found bool
	for _, s := range spans {
		if s.Name == "mlit.FetchLocationOptimization" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("span 'mlit.FetchLocationOptimization' not found; got: %v", spanNames(spans))
	}
}

func TestFetchLocationOptimization_CacheHitAttribute(t *testing.T) {
	exp := setupTestTracer(t)

	resp := LocationOptimizationGeoJSON{Type: "FeatureCollection", Features: []LocationOptimizationFeature{}}
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(makeGeoJSONHandler(body))
	defer srv.Close()

	client := newOtelTestClient(srv)
	// First call — cache miss.
	_, err := client.FetchLocationOptimization(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	exp.Reset()

	// Second call — cache hit.
	_, err = client.FetchLocationOptimization(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	spans := exp.GetSpans()
	var cacheHit *bool
	for _, s := range spans {
		if s.Name != "mlit.FetchLocationOptimization" {
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

func TestFetchLocationOptimization_ErrorSpanStatus(t *testing.T) {
	exp := setupTestTracer(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newOtelTestClient(srv)
	_, err := client.FetchLocationOptimization(context.Background(), 14, 14547, 6451)
	if err == nil {
		t.Fatal("expected error from 500 server, got nil")
	}

	spans := exp.GetSpans()
	var foundError bool
	for _, s := range spans {
		if s.Name == "mlit.FetchLocationOptimization" && s.Status.Code.String() == "Error" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("expected span with Error status; spans: %v", spanNames(spans))
	}
}

// ---- FetchEmbankment (XKT020) ----

func TestFetchEmbankment_SpanCreated(t *testing.T) {
	exp := setupTestTracer(t)

	resp := EmbankmentGeoJSON{
		Type: "FeatureCollection",
		Features: []EmbankmentFeature{
			{Properties: EmbankmentProperties{EmbankmentClassification: "谷埋め型"}},
		},
	}
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(makeGeoJSONHandler(body))
	defer srv.Close()

	client := newOtelTestClient(srv)
	_, err := client.FetchEmbankment(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("FetchEmbankment returned unexpected error: %v", err)
	}

	spans := exp.GetSpans()
	var found bool
	for _, s := range spans {
		if s.Name == "mlit.FetchEmbankment" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("span 'mlit.FetchEmbankment' not found; got: %v", spanNames(spans))
	}
}

func TestFetchEmbankment_CacheHitAttribute(t *testing.T) {
	exp := setupTestTracer(t)

	resp := EmbankmentGeoJSON{Type: "FeatureCollection", Features: []EmbankmentFeature{}}
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(makeGeoJSONHandler(body))
	defer srv.Close()

	client := newOtelTestClient(srv)
	// First call — cache miss.
	_, err := client.FetchEmbankment(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	exp.Reset()

	// Second call — cache hit.
	_, err = client.FetchEmbankment(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	spans := exp.GetSpans()
	var cacheHit *bool
	for _, s := range spans {
		if s.Name != "mlit.FetchEmbankment" {
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

func TestFetchEmbankment_ErrorSpanStatus(t *testing.T) {
	exp := setupTestTracer(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newOtelTestClient(srv)
	_, err := client.FetchEmbankment(context.Background(), 14, 14547, 6451)
	if err == nil {
		t.Fatal("expected error from 500 server, got nil")
	}

	spans := exp.GetSpans()
	var foundError bool
	for _, s := range spans {
		if s.Name == "mlit.FetchEmbankment" && s.Status.Code.String() == "Error" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("expected span with Error status; spans: %v", spanNames(spans))
	}
}

// ---- FetchUrbanRoad (XKT030) ----

func TestFetchUrbanRoad_SpanCreated(t *testing.T) {
	exp := setupTestTracer(t)

	resp := UrbanRoadGeoJSON{
		Type: "FeatureCollection",
		Features: []UrbanRoadFeature{
			{Properties: UrbanRoadProperties{PlanningRoadJa: "都市計画道路A", KubunID: 3011}},
		},
	}
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(makeGeoJSONHandler(body))
	defer srv.Close()

	client := newOtelTestClient(srv)
	_, err := client.FetchUrbanRoad(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("FetchUrbanRoad returned unexpected error: %v", err)
	}

	spans := exp.GetSpans()
	var found bool
	for _, s := range spans {
		if s.Name == "mlit.FetchUrbanRoad" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("span 'mlit.FetchUrbanRoad' not found; got: %v", spanNames(spans))
	}
}

func TestFetchUrbanRoad_CacheHitAttribute(t *testing.T) {
	exp := setupTestTracer(t)

	resp := UrbanRoadGeoJSON{Type: "FeatureCollection", Features: []UrbanRoadFeature{}}
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(makeGeoJSONHandler(body))
	defer srv.Close()

	client := newOtelTestClient(srv)
	// First call — cache miss.
	_, err := client.FetchUrbanRoad(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	exp.Reset()

	// Second call — cache hit.
	_, err = client.FetchUrbanRoad(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	spans := exp.GetSpans()
	var cacheHit *bool
	for _, s := range spans {
		if s.Name != "mlit.FetchUrbanRoad" {
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

func TestFetchUrbanRoad_ErrorSpanStatus(t *testing.T) {
	exp := setupTestTracer(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newOtelTestClient(srv)
	_, err := client.FetchUrbanRoad(context.Background(), 14, 14547, 6451)
	if err == nil {
		t.Fatal("expected error from 500 server, got nil")
	}

	spans := exp.GetSpans()
	var foundError bool
	for _, s := range spans {
		if s.Name == "mlit.FetchUrbanRoad" && s.Status.Code.String() == "Error" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("expected span with Error status; spans: %v", spanNames(spans))
	}
}

// ---- FetchDisasterHistory (XST001) ----

func TestFetchDisasterHistory_SpanCreated(t *testing.T) {
	exp := setupTestTracer(t)

	resp := DisasterHistoryGeoJSON{
		Type: "FeatureCollection",
		Features: []DisasterHistoryFeature{
			{Properties: DisasterHistoryProperties{DisasterNameJa: "浸水域", DisasterDate: "20190101"}},
		},
	}
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(makeGeoJSONHandler(body))
	defer srv.Close()

	client := newOtelTestClient(srv)
	_, err := client.FetchDisasterHistory(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("FetchDisasterHistory returned unexpected error: %v", err)
	}

	spans := exp.GetSpans()
	var found bool
	for _, s := range spans {
		if s.Name == "mlit.FetchDisasterHistory" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("span 'mlit.FetchDisasterHistory' not found; got: %v", spanNames(spans))
	}
}

func TestFetchDisasterHistory_CacheHitAttribute(t *testing.T) {
	exp := setupTestTracer(t)

	resp := DisasterHistoryGeoJSON{Type: "FeatureCollection", Features: []DisasterHistoryFeature{}}
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(makeGeoJSONHandler(body))
	defer srv.Close()

	client := newOtelTestClient(srv)
	// First call — cache miss.
	_, err := client.FetchDisasterHistory(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	exp.Reset()

	// Second call — cache hit.
	_, err = client.FetchDisasterHistory(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	spans := exp.GetSpans()
	var cacheHit *bool
	for _, s := range spans {
		if s.Name != "mlit.FetchDisasterHistory" {
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

func TestFetchDisasterHistory_ErrorSpanStatus(t *testing.T) {
	exp := setupTestTracer(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newOtelTestClient(srv)
	_, err := client.FetchDisasterHistory(context.Background(), 14, 14547, 6451)
	if err == nil {
		t.Fatal("expected error from 500 server, got nil")
	}

	spans := exp.GetSpans()
	var foundError bool
	for _, s := range spans {
		if s.Name == "mlit.FetchDisasterHistory" && s.Status.Code.String() == "Error" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("expected span with Error status; spans: %v", spanNames(spans))
	}
}
