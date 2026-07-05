package api

// rent 系ハンドラの HTTP バインディングテスト。
// パラメータ検証と JSON null 契約を検証し、
// 期間計算・信頼度判定ロジックは service/rent_test.go で検証する。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

func TestGetRentStats_MissingArea(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/rent-stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetRentStats_Success(t *testing.T) {
	client := &mockMLITClient{
		rentStatsFunc: func(_ context.Context, _ mlit.LandPriceQuery, _ float64) (domain.RentStatsResult, error) {
			return domain.RentStatsResult{Median: 80000, Average: 82000, Count: 15}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/rent-stats?area=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result domain.RentStatsResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Median != 80000 {
		t.Errorf("expected median=80000, got %v", result.Median)
	}
	if result.Average != 82000 {
		t.Errorf("expected average=82000, got %v", result.Average)
	}
	if result.Count != 15 {
		t.Errorf("expected count=15, got %d", result.Count)
	}
}

func TestGetRentStats_NoData(t *testing.T) {
	client := &mockMLITClient{
		rentStatsFunc: func(_ context.Context, _ mlit.LandPriceQuery, _ float64) (domain.RentStatsResult, error) {
			return domain.RentStatsResult{}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/rent-stats?area=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.TrimSpace(body) != "null" {
		t.Errorf("expected null response for count=0, got %s", body)
	}
}

func TestGetRentStats_FetchError(t *testing.T) {
	client := &mockMLITClient{
		rentStatsFunc: func(_ context.Context, _ mlit.LandPriceQuery, _ float64) (domain.RentStatsResult, error) {
			return domain.RentStatsResult{}, errors.New("upstream error")
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/rent-stats?area=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// エラーはサイレントに null を返す
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (silent error), got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.TrimSpace(body) != "null" {
		t.Errorf("expected null response on error, got %s", body)
	}
}
