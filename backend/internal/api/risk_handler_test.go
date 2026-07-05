package api

// risk / hazard 系ハンドラの HTTP バインディングテスト。
// パラメータ検証・レスポンス形状のみを検証し、
// リスク構築ロジックは service/risk_test.go で検証する。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yield-guard/backend/internal/domain"
)

// ---- GetUrbanRisks (/api/urban-risks) ----

func TestGetUrbanRisks_MissingParams(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	for _, url := range []string{
		"/api/urban-risks",
		"/api/urban-risks?lat=35.68",
		"/api/urban-risks?lng=139.69",
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code != http.StatusBadRequest {
			t.Errorf("url=%s: expected 400, got %d", url, w.Code)
		}
	}
}

func TestGetUrbanRisks_InvalidRange(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	for _, url := range []string{
		"/api/urban-risks?lat=10&lng=139.69",  // lat 範囲外（< 20）
		"/api/urban-risks?lat=35.68&lng=200",  // lng 範囲外（> 154）
		"/api/urban-risks?lat=abc&lng=139.69", // 数値でない
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code != http.StatusBadRequest {
			t.Errorf("url=%s: expected 400, got %d", url, w.Code)
		}
	}
}

func TestGetUrbanRisks_EmptyResult(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/urban-risks?lat=35.68&lng=139.69", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var risks []domain.UrbanRisk
	if err := json.NewDecoder(w.Body).Decode(&risks); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(risks) != 0 {
		t.Errorf("expected empty array, got %d risks", len(risks))
	}
}

// ---- GetHazardInfo (/api/hazard) ----

func TestGetHazardInfo_MissingLatLng(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	for _, url := range []string{"/api/hazard", "/api/hazard?lat=35.68", "/api/hazard?lng=139.69"} {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("url=%s: expected 400, got %d", url, w.Code)
		}
	}
}

func TestGetHazardInfo_InvalidLatLng(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	cases := []string{
		"/api/hazard?lat=999&lng=139.69", // lat 範囲外
		"/api/hazard?lat=35.68&lng=200",  // lng 範囲外
		"/api/hazard?lat=abc&lng=139.69", // lat 非数値
	}
	for _, url := range cases {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("url=%s: expected 400, got %d", url, w.Code)
		}
	}
}

func TestGetHazardInfo_EmptyResult(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/hazard?lat=35.68&lng=139.69", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var risks []domain.UrbanRisk
	if err := json.NewDecoder(w.Body).Decode(&risks); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	// ハザードなし → 空スライス（null でなく []）
	if risks == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(risks) != 0 {
		t.Errorf("expected 0 risks, got %d", len(risks))
	}
}
