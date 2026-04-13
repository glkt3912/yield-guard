package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockMLITClient は MLITClient インターフェースのテスト用モック
type mockMLITClient struct {
	fetchFunc func(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
}

func (m *mockMLITClient) FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
	if m.fetchFunc == nil {
		panic(fmt.Sprintf("mockMLITClient.FetchLandPrices called unexpectedly (fetchFunc is nil)"))
	}
	return m.fetchFunc(ctx, q)
}

// newTestRouter はモッククライアントを使ったテスト用ルーターを返す
func newTestRouter(client MLITClient) *gin.Engine {
	h := NewHandler(client)
	return NewRouter(h)
}

var validBase = domain.InvestmentInput{
	LandPrice:       5_000_000,
	BuildingCost:    10_000_000,
	MonthlyRent:     100_000,
	VacancyRate:     0.05,
	LoanAmount:      0,
	AnnualLoanRate:  0.015,
	LoanYears:       35,
	MiscExpenseRate: 0.07,
	ExpenseRate:     0.20,
	IncomeTaxRate:   0.33,
	ExitYieldTarget: 0.06,
	HoldingYears:    10,
}

func withField(base domain.InvestmentInput, apply func(*domain.InvestmentInput)) domain.InvestmentInput {
	apply(&base)
	return base
}

func TestValidateInvestmentInput_Boundaries(t *testing.T) {
	tests := []struct {
		name    string
		input   domain.InvestmentInput
		wantErr bool
	}{
		// LandPrice
		{"landPrice=0 → error", withField(validBase, func(i *domain.InvestmentInput) { i.LandPrice = 0 }), true},
		{"landPrice=1 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.LandPrice = 1 }), false},
		{"landPrice=10_000_000_000 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.LandPrice = 10_000_000_000 }), false},
		{"landPrice=10_000_000_001 → error", withField(validBase, func(i *domain.InvestmentInput) { i.LandPrice = 10_000_000_001 }), true},
		// BuildingCost
		{"buildingCost=0 → error", withField(validBase, func(i *domain.InvestmentInput) { i.BuildingCost = 0 }), true},
		{"buildingCost=1 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.BuildingCost = 1 }), false},
		{"buildingCost=10_000_000_001 → error", withField(validBase, func(i *domain.InvestmentInput) { i.BuildingCost = 10_000_000_001 }), true},
		// MonthlyRent
		{"monthlyRent=0 → error", withField(validBase, func(i *domain.InvestmentInput) { i.MonthlyRent = 0 }), true},
		{"monthlyRent=1 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.MonthlyRent = 1 }), false},
		// VacancyRate
		{"vacancyRate=-0.01 → error", withField(validBase, func(i *domain.InvestmentInput) { i.VacancyRate = -0.01 }), true},
		{"vacancyRate=0 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.VacancyRate = 0 }), false},
		{"vacancyRate=0.99 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.VacancyRate = 0.99 }), false},
		{"vacancyRate=1.0 → error", withField(validBase, func(i *domain.InvestmentInput) { i.VacancyRate = 1.0 }), true},
		// LoanAmount
		{"loanAmount=-1 → error", withField(validBase, func(i *domain.InvestmentInput) { i.LoanAmount = -1 }), true},
		{"loanAmount=0 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.LoanAmount = 0 }), false},
		// AnnualLoanRate
		{"annualLoanRate=-0.001 → error", withField(validBase, func(i *domain.InvestmentInput) { i.AnnualLoanRate = -0.001 }), true},
		{"annualLoanRate=0 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.AnnualLoanRate = 0 }), false},
		{"annualLoanRate=0.3 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.AnnualLoanRate = 0.3 }), false},
		{"annualLoanRate=0.301 → error", withField(validBase, func(i *domain.InvestmentInput) { i.AnnualLoanRate = 0.301 }), true},
		// LoanYears
		{"loanYears=-1 → error", withField(validBase, func(i *domain.InvestmentInput) { i.LoanYears = -1 }), true},
		{"loanYears=0 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.LoanYears = 0 }), false},
		{"loanYears=50 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.LoanYears = 50 }), false},
		{"loanYears=51 → error", withField(validBase, func(i *domain.InvestmentInput) { i.LoanYears = 51 }), true},
		// MiscExpenseRate
		{"miscExpenseRate=-0.01 → error", withField(validBase, func(i *domain.InvestmentInput) { i.MiscExpenseRate = -0.01 }), true},
		{"miscExpenseRate=0 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.MiscExpenseRate = 0 }), false},
		{"miscExpenseRate=0.5 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.MiscExpenseRate = 0.5 }), false},
		{"miscExpenseRate=0.51 → error", withField(validBase, func(i *domain.InvestmentInput) { i.MiscExpenseRate = 0.51 }), true},
		// ExpenseRate
		{"expenseRate=-0.01 → error", withField(validBase, func(i *domain.InvestmentInput) { i.ExpenseRate = -0.01 }), true},
		{"expenseRate=0 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.ExpenseRate = 0 }), false},
		{"expenseRate=0.9 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.ExpenseRate = 0.9 }), false},
		{"expenseRate=0.91 → error", withField(validBase, func(i *domain.InvestmentInput) { i.ExpenseRate = 0.91 }), true},
		// IncomeTaxRate
		{"incomeTaxRate=-0.01 → error", withField(validBase, func(i *domain.InvestmentInput) { i.IncomeTaxRate = -0.01 }), true},
		{"incomeTaxRate=0 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.IncomeTaxRate = 0 }), false},
		{"incomeTaxRate=0.6 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.IncomeTaxRate = 0.6 }), false},
		{"incomeTaxRate=0.61 → error", withField(validBase, func(i *domain.InvestmentInput) { i.IncomeTaxRate = 0.61 }), true},
		// ExitYieldTarget
		{"exitYieldTarget=0 → error", withField(validBase, func(i *domain.InvestmentInput) { i.ExitYieldTarget = 0 }), true},
		{"exitYieldTarget=0.001 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.ExitYieldTarget = 0.001 }), false},
		{"exitYieldTarget=0.5 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.ExitYieldTarget = 0.5 }), false},
		{"exitYieldTarget=0.51 → error", withField(validBase, func(i *domain.InvestmentInput) { i.ExitYieldTarget = 0.51 }), true},
		// HoldingYears
		{"holdingYears=-1 → error", withField(validBase, func(i *domain.InvestmentInput) { i.HoldingYears = -1 }), true},
		{"holdingYears=0 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.HoldingYears = 0 }), false},
		{"holdingYears=50 → ok", withField(validBase, func(i *domain.InvestmentInput) { i.HoldingYears = 50 }), false},
		{"holdingYears=51 → error", withField(validBase, func(i *domain.InvestmentInput) { i.HoldingYears = 51 }), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInvestmentInput(tt.input)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestAnalyze_ValidInput(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})

	body, _ := json.Marshal(validBase)
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewReader(body))
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
	r := newTestRouter(&mockMLITClient{})

	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAnalyze_ValidationError(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})

	invalid := validBase
	invalid.LandPrice = -1
	body, _ := json.Marshal(invalid)
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", bytes.NewReader(body))
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

func TestGetLandPrices_MissingArea(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices?year=2024&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandPrices_InvalidYear(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices?area=13&year=2000&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandPrices_InvalidQuarter(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices?area=13&year=2024&quarter=5&to_year=2024&to_quarter=4", nil)
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
	r := newTestRouter(client)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices?area=13&year=2024&quarter=1&to_year=2024&to_quarter=4", nil)
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
	r := newTestRouter(client)
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices?area=13&year=2024&quarter=1&to_year=2024&to_quarter=4", nil)
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

func TestCompareLandPrice_MissingPrice(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/compare?area=13&year=2024&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompareLandPrice_InvalidPrice(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
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
	r := newTestRouter(client)
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

func TestHealthCheck(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", resp["status"])
	}
}
