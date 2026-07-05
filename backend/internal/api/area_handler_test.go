package api

// area 系ハンドラの HTTP バインディングテスト。
// パラメータ検証・ステータスコード変換のみを検証し、
// ユースケースロジックは service/area_discovery_test.go で検証する。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// ---- HandleAreaDiscovery (/api/area-discovery) ----

func TestHandleAreaDiscovery_MissingPrefecture(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/area-discovery", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected error message in response body")
	}
}

func TestHandleAreaDiscovery_MunicipalityFetchError(t *testing.T) {
	client := &mockMLITClient{
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return nil, errors.New("upstream error")
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/area-discovery?prefecture=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// HandleAreaDiscovery returns 500 on municipality fetch error
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAreaDiscovery_Success(t *testing.T) {
	client := &mockMLITClient{
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return []mlit.Municipality{
				{ID: "13101", Name: "千代田区"},
				{ID: "13102", Name: "中央区"},
			}, nil
		},
		fetchFunc: func(_ context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{Period: "2024年第1四半期", TradePrice: 10_000_000, Area: 100, PricePerSqm: 100_000, PricePerTsubo: 330_578},
				{Period: "2024年第2四半期", TradePrice: 11_000_000, Area: 110, PricePerSqm: 100_000, PricePerTsubo: 330_578},
				{Period: "2024年第3四半期", TradePrice: 12_000_000, Area: 120, PricePerSqm: 100_000, PricePerTsubo: 330_578},
			}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/area-discovery?prefecture=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.AreaDiscoveryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Prefecture != "13" {
		t.Errorf("expected prefecture=13, got %q", resp.Prefecture)
	}
	if len(resp.Items) == 0 {
		t.Error("expected non-empty items")
	} else {
		// The mock returns municipalities "13101" and "13102"; verify the first item
		// has a recognizable municipality code from the mock
		found := false
		for _, item := range resp.Items {
			if item.MunicipalityCode == "13101" || item.MunicipalityCode == "13102" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected item with MunicipalityCode 13101 or 13102, got %+v", resp.Items)
		}
		// The mock transactions all have PricePerTsubo=330578, so MedianTsubo should be non-zero
		if resp.Items[0].MedianTsubo == 0 {
			t.Errorf("expected non-zero MedianTsubo, got %f", resp.Items[0].MedianTsubo)
		}
	}
}

// ---- HandleAreaSummary (/api/area-discovery/summary) ----

func TestHandleAreaSummary_MissingParams(t *testing.T) {
	cases := []struct {
		url string
	}{
		{"/api/area-discovery/summary"},
		{"/api/area-discovery/summary?area=13"},
		{"/api/area-discovery/summary?municipality=13101"},
	}
	for _, tc := range cases {
		r := newTestRouter(&mockMLITClient{}, nil)
		req := httptest.NewRequest(http.MethodGet, tc.url, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("url=%s: expected 400, got %d: %s", tc.url, w.Code, w.Body.String())
		}
		var resp map[string]string
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if resp["error"] == "" {
			t.Errorf("url=%s: expected error message in response body", tc.url)
		}
	}
}

func TestHandleAreaSummary_Success(t *testing.T) {
	client := &mockMLITClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{Period: "2024年第1四半期", TradePrice: 5_000_000, Area: 50, PricePerSqm: 100_000, PricePerTsubo: 100_000},
				{Period: "2024年第2四半期", TradePrice: 5_500_000, Area: 55, PricePerSqm: 100_000, PricePerTsubo: 100_000},
				{Period: "2024年第3四半期", TradePrice: 6_000_000, Area: 60, PricePerSqm: 100_000, PricePerTsubo: 100_000},
			}, nil
		},
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return []mlit.Municipality{
				{ID: "13101", Name: "千代田区"},
			}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/area-discovery/summary?area=13&municipality=13101", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["summary"]; !ok {
		t.Error("expected 'summary' key in response")
	}
	// PricePerTsubo=100_000 → CalcYieldDifficulty → "達成可能"
	// noopSummarizer returns "" → fallback to YieldDifficultyLabel
	if resp["summary"] != "達成可能" {
		t.Errorf("expected summary=%q, got %q", "達成可能", resp["summary"])
	}
}
