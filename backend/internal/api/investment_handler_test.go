package api

// investment 系ハンドラの HTTP バインディングテスト。
// 入力検証・ステータスコード変換を検証し、
// 賃料下落率ヒントの傾向判定ロジックは service/investment_test.go で検証する。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yield-guard/backend/internal/domain"
)

// ---- Analyze (/api/investment/analyze) ----

func TestAnalyze_ValidInput(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)

	body, _ := json.Marshal(validBase)
	req := httptest.NewRequest(http.MethodPost, "/api/investment/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result domain.InvestmentResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.TotalInvestment <= 0 {
		t.Error("expected totalInvestment > 0")
	}
}

func TestAnalyze_InvalidJSON(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/investment/analyze", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAnalyze_ValidationError(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)

	invalid := validBase
	invalid.LandPrice = -1
	body, _ := json.Marshal(invalid)
	req := httptest.NewRequest(http.MethodPost, "/api/investment/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

// ---- MonteCarlo (/api/investment/simulate) ----

func TestMonteCarlo_ValidInput(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)

	input := domain.MonteCarloInput{
		Base:        validBase,
		Simulations: 100,
	}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/investment/simulate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result domain.MonteCarloResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.SimulationCount != 100 {
		t.Errorf("expected simulationCount=100, got %d", result.SimulationCount)
	}
	if result.SuccessRate < 0 || result.SuccessRate > 1 {
		t.Errorf("successRate out of range: %f", result.SuccessRate)
	}
}

func TestMonteCarlo_InvalidJSON(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/investment/simulate", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected error message in response body")
	}
}

func TestMonteCarlo_ValidationError(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)

	invalid := validBase
	invalid.LandPrice = -1 // invalid
	input := domain.MonteCarloInput{
		Base:        invalid,
		Simulations: 100,
	}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/investment/simulate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

// ---- GetRentDeclineHint (/api/investment/rent-decline-hint) ----

func TestGetRentDeclineHint_MissingArea(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/investment/rent-decline-hint", nil)
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

func TestGetRentDeclineHint_AllYearsAPIError(t *testing.T) {
	client := &mockMLITClient{
		appraisalFunc: func(_ context.Context, _, _ string, _ int, _ string) ([]domain.LandAppraisalItem, error) {
			return nil, errors.New("upstream error")
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/investment/rent-decline-hint?area=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetRentDeclineHint_PartialYearError(t *testing.T) {
	// 2022, 2023 はエラー（5年中2年）、2024〜2026 は有効な下落データを返す
	client := &mockMLITClient{
		appraisalFunc: func(_ context.Context, _, _ string, year int, _ string) ([]domain.LandAppraisalItem, error) {
			switch year {
			case 2022, 2023:
				return nil, errors.New("upstream error")
			case 2024:
				return []domain.LandAppraisalItem{
					{Year: 2024, PricePerSqm: 180000, ChangeRate: -0.04},
					{Year: 2024, PricePerSqm: 185000, ChangeRate: -0.04},
					{Year: 2024, PricePerSqm: 175000, ChangeRate: -0.04},
					{Year: 2024, PricePerSqm: 182000, ChangeRate: -0.04},
					{Year: 2024, PricePerSqm: 178000, ChangeRate: -0.04},
				}, nil
			case 2025:
				return []domain.LandAppraisalItem{
					{Year: 2025, PricePerSqm: 170000, ChangeRate: -0.03},
					{Year: 2025, PricePerSqm: 175000, ChangeRate: -0.03},
					{Year: 2025, PricePerSqm: 168000, ChangeRate: -0.03},
					{Year: 2025, PricePerSqm: 172000, ChangeRate: -0.03},
					{Year: 2025, PricePerSqm: 165000, ChangeRate: -0.03},
				}, nil
			case 2026:
				return []domain.LandAppraisalItem{
					{Year: 2026, PricePerSqm: 162000, ChangeRate: -0.02},
					{Year: 2026, PricePerSqm: 165000, ChangeRate: -0.02},
					{Year: 2026, PricePerSqm: 160000, ChangeRate: -0.02},
					{Year: 2026, PricePerSqm: 163000, ChangeRate: -0.02},
					{Year: 2026, PricePerSqm: 158000, ChangeRate: -0.02},
				}, nil
			default:
				return []domain.LandAppraisalItem{}, nil
			}
		},
	}
	r := newTestRouter(client, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/investment/rent-decline-hint?area=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 一部年エラーでも有効データがあれば200を返す
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for partial year error, got %d: %s", w.Code, w.Body.String())
	}
	var hint domain.RentDeclineHint
	if err := json.NewDecoder(w.Body).Decode(&hint); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if hint.Basis != "land_appraisal" {
		t.Errorf("expected basis=land_appraisal, got %q", hint.Basis)
	}
}
