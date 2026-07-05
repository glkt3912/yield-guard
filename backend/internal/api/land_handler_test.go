package api

// land 系ハンドラの HTTP バインディングテスト。
// パラメータ検証・ステータスコード変換のみを検証し、
// 統計・比較ロジックは service/land_price_test.go で検証する。

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

// ---- GetLandPrices (/api/land-prices/stats) ----

func TestGetLandPrices_MissingArea(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/stats?year=2024&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandPrices_InvalidYear(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/stats?area=13&year=2000&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandPrices_InvalidQuarter(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/stats?area=13&year=2024&quarter=5&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandPrices_APIError(t *testing.T) {
	client := &mockMLITClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return nil, errors.New("upstream error")
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/stats?area=13&year=2024&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestGetLandPrices_Success(t *testing.T) {
	client := &mockMLITClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{Period: "2024年第1四半期", TradePrice: 10_000_000, Area: 100, PricePerSqm: 100_000, PricePerTsubo: 330_578},
			}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/stats?area=13&year=2024&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var stats domain.LandPriceStats
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if stats.Count != 1 {
		t.Errorf("expected count=1, got %d", stats.Count)
	}
}

// ---- CompareLandPrice (/api/land-prices/compare) ----

func TestCompareLandPrice_MissingPrice(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/compare?area=13&year=2024&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompareLandPrice_InvalidPrice(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/compare?area=13&year=2024&quarter=1&to_year=2024&to_quarter=4&price=-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompareLandPrice_Success(t *testing.T) {
	client := &mockMLITClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{Period: "2024年第1四半期", TradePrice: 10_000_000, Area: 100, PricePerSqm: 100_000, PricePerTsubo: 330_578},
			}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/compare?area=13&year=2024&quarter=1&to_year=2024&to_quarter=4&price=10000000&area_sqm=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var comparison domain.LandPriceComparison
	if err := json.NewDecoder(w.Body).Decode(&comparison); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if comparison.Assessment == "" {
		t.Error("expected non-empty assessment")
	}
}

// ---- EstimateLandPrice (/api/land-prices/estimate) ----

func TestEstimateLandPrice_InvalidRidershipScore(t *testing.T) {
	client := &mockMLITClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{PricePerTsubo: 200_000, BuildingYear: 2010, StationMinutes: 10},
			}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet,
		"/api/land-prices/estimate?area=10&year=2024&quarter=1&to_year=2024&to_quarter=4&price=5000000&area_sqm=100&building_age=10&ridership_score=Z",
		nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid ridership_score, got %d", w.Code)
	}
}

// 建築年データなし → service.ErrEstimateDataInsufficient → 422
func TestEstimateLandPrice_DataInsufficient(t *testing.T) {
	client := &mockMLITClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{PricePerTsubo: 200_000}, // BuildingYear なし
			}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet,
		"/api/land-prices/estimate?area=10&year=2024&quarter=1&to_year=2024&to_quarter=4&price=5000000&area_sqm=100&building_age=10",
		nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for insufficient data, got %d: %s", w.Code, w.Body.String())
	}
}

// ---- GetMunicipalities (/api/municipalities) ----

func TestGetMunicipalities_MissingArea(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/municipalities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetMunicipalities_APIError(t *testing.T) {
	client := &mockMLITClient{
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return nil, errors.New("upstream error")
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/municipalities?area=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestGetMunicipalities_Success(t *testing.T) {
	client := &mockMLITClient{
		muniFunc: func(_ context.Context, area string) ([]mlit.Municipality, error) {
			return []mlit.Municipality{
				{ID: "13101", Name: "千代田区"},
				{ID: "13102", Name: "中央区"},
			}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/municipalities?area=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result []mlit.Municipality
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 municipalities, got %d", len(result))
	}
	if result[0].ID != "13101" || result[0].Name != "千代田区" {
		t.Errorf("unexpected first entry: %+v", result[0])
	}
}

// ---- GetLandAppraisals (/api/land-appraisals) ----

func TestGetLandAppraisals_MissingArea(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-appraisals?year=2024", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandAppraisals_InvalidYear(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-appraisals?area=13&year=2010", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandAppraisals_Success(t *testing.T) {
	client := &mockMLITClient{
		appraisalFunc: func(_ context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error) {
			return []domain.LandAppraisalItem{
				{Year: 2024, PricePerSqm: 1_000_000, ChangeRate: 0.03, District: "千代田"},
				{Year: 2024, PricePerSqm: 800_000, ChangeRate: 0.02, District: "中央"},
			}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-appraisals?area=13&year=2024", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.AppraisalComparisonResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.AppraisalCount != 2 {
		t.Errorf("AppraisalCount = %d, want 2", resp.AppraisalCount)
	}
	if resp.AppraisalMedianPerSqm != 900_000 {
		t.Errorf("AppraisalMedianPerSqm = %v, want 900000", resp.AppraisalMedianPerSqm)
	}
}

func TestGetLandAppraisals_NoData(t *testing.T) {
	client := &mockMLITClient{
		appraisalFunc: func(_ context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error) {
			return []domain.LandAppraisalItem{}, nil
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-appraisals?area=13&year=2024", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestGetLandAppraisals_UpstreamError(t *testing.T) {
	client := &mockMLITClient{
		appraisalFunc: func(_ context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error) {
			return nil, errors.New("upstream error")
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-appraisals?area=13&year=2024", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestGetLandAppraisals_InvalidDivision(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/land-appraisals?area=13&year=2024&division=99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
