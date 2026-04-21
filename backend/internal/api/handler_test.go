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
	fetchFunc            func(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
	muniFunc             func(ctx context.Context, area string) ([]mlit.Municipality, error)
	ridershipFunc        func(ctx context.Context, z, x, y int) ([]mlit.StationRidership, error)
	populationFunc       func(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error)
	appraisalFunc        func(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error)
	locationOptFunc      func(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error)
	embankmentFunc       func(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error)
	urbanRoadFunc        func(ctx context.Context, z, x, y int) ([]domain.UrbanRoadItem, error)
	disasterHistoryFunc  func(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error)
}

func (m *mockMLITClient) FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
	if m.fetchFunc == nil {
		panic("mockMLITClient.FetchLandPrices called unexpectedly (fetchFunc is nil)")
	}
	return m.fetchFunc(ctx, q)
}

func (m *mockMLITClient) FetchMunicipalities(ctx context.Context, area string) ([]mlit.Municipality, error) {
	if m.muniFunc == nil {
		return []mlit.Municipality{}, nil
	}
	return m.muniFunc(ctx, area)
}

func (m *mockMLITClient) FetchStationRidership(ctx context.Context, z, x, y int) ([]mlit.StationRidership, error) {
	if m.ridershipFunc == nil {
		return []mlit.StationRidership{}, nil
	}
	return m.ridershipFunc(ctx, z, x, y)
}

func (m *mockMLITClient) FetchPopulationForecast(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error) {
	if m.populationFunc == nil {
		return []domain.PopulationForecastItem{}, nil
	}
	return m.populationFunc(ctx, z, x, y)
}

func (m *mockMLITClient) FetchLandAppraisals(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error) {
	if m.appraisalFunc == nil {
		return []domain.LandAppraisalItem{}, nil
	}
	return m.appraisalFunc(ctx, area, city, year, division)
}

func (m *mockMLITClient) FetchLocationOptimization(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error) {
	if m.locationOptFunc != nil {
		return m.locationOptFunc(ctx, z, x, y)
	}
	return []domain.LocationOptimizationItem{}, nil
}

func (m *mockMLITClient) FetchEmbankment(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error) {
	if m.embankmentFunc != nil {
		return m.embankmentFunc(ctx, z, x, y)
	}
	return []domain.EmbankmentItem{}, nil
}

func (m *mockMLITClient) FetchUrbanRoad(ctx context.Context, z, x, y int) ([]domain.UrbanRoadItem, error) {
	if m.urbanRoadFunc != nil {
		return m.urbanRoadFunc(ctx, z, x, y)
	}
	return []domain.UrbanRoadItem{}, nil
}

func (m *mockMLITClient) FetchDisasterHistory(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error) {
	if m.disasterHistoryFunc != nil {
		return m.disasterHistoryFunc(ctx, z, x, y)
	}
	return []domain.DisasterHistoryItem{}, nil
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
	r := newTestRouter(&mockMLITClient{})

	req := httptest.NewRequest(http.MethodPost, "/api/investment/analyze", bytes.NewBufferString("not-json"))
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

func TestGetLandPrices_MissingArea(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/stats?year=2024&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandPrices_InvalidYear(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/land-prices/stats?area=13&year=2000&quarter=1&to_year=2024&to_quarter=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandPrices_InvalidQuarter(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
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
	r := newTestRouter(client)
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
	r := newTestRouter(client)
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

func TestGetMunicipalities_MissingArea(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
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
	r := newTestRouter(client)
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
	r := newTestRouter(client)
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

func TestGetStationRidership_MissingLatLng(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/station-ridership", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetStationRidership_InvalidLatLng(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/station-ridership?lat=999&lng=139.6503", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetStationRidership_InvalidZ(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/station-ridership?lat=35.6762&lng=139.6503&z=16", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetStationRidership_Success(t *testing.T) {
	client := &mockMLITClient{
		ridershipFunc: func(_ context.Context, z, x, y int) ([]mlit.StationRidership, error) {
			// lat=35.6762, lng=139.6503, z=14 → x=14547, y=6451
			if z != 14 || x != 14547 || y != 6451 {
				t.Errorf("unexpected tile: z=%d x=%d y=%d", z, x, y)
			}
			return []mlit.StationRidership{
				{StationName: "渋谷", LineName: "JR山手線", Passengers: 360000},
				{StationName: "代々木上原", LineName: "小田急小田原線", Passengers: 85000},
			}, nil
		},
	}
	r := newTestRouter(client)
	req := httptest.NewRequest(http.MethodGet, "/api/station-ridership?lat=35.6762&lng=139.6503", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result []domain.StationRidershipResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 stations, got %d", len(result))
	}
	if result[0].StationName != "渋谷" {
		t.Errorf("unexpected station: %+v", result[0])
	}
	if result[0].Passengers != 360000 {
		t.Errorf("expected 360000 passengers, got %d", result[0].Passengers)
	}
	if result[0].DemandScore != domain.RidershipScoreA {
		t.Errorf("expected score A, got %s", result[0].DemandScore)
	}
}

func TestGetStationRidership_APIError(t *testing.T) {
	client := &mockMLITClient{
		ridershipFunc: func(_ context.Context, _, _, _ int) ([]mlit.StationRidership, error) {
			return nil, fmt.Errorf("upstream error")
		},
	}
	r := newTestRouter(client)
	req := httptest.NewRequest(http.MethodGet, "/api/station-ridership?lat=35.6762&lng=139.6503", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestEstimateLandPrice_InvalidRidershipScore(t *testing.T) {
	client := &mockMLITClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{PricePerTsubo: 200_000, BuildingYear: 2010, StationMinutes: 10},
			}, nil
		},
	}
	r := newTestRouter(client)
	req := httptest.NewRequest(http.MethodGet,
		"/api/land-prices/estimate?area=10&year=2024&quarter=1&to_year=2024&to_quarter=4&price=5000000&area_sqm=100&building_age=10&ridership_score=Z",
		nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid ridership_score, got %d", w.Code)
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

func TestGetPopulationForecast_MissingLatLng(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/population-forecast", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetPopulationForecast_Success(t *testing.T) {
	client := &mockMLITClient{
		populationFunc: func(_ context.Context, z, x, y int) ([]domain.PopulationForecastItem, error) {
			return []domain.PopulationForecastItem{
				{Year: 2020, Pop: 1000},
				{Year: 2025, Pop: 950},
				{Year: 2030, Pop: 900},
				{Year: 2035, Pop: 850},
				{Year: 2040, Pop: 800},
				{Year: 2045, Pop: 760},
				{Year: 2050, Pop: 720},
			}, nil
		},
	}
	r := newTestRouter(client)
	req := httptest.NewRequest(http.MethodGet, "/api/population-forecast?lat=35.6762&lng=139.6503", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.PopulationForecastResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ChangeRate30yr >= 0 {
		t.Errorf("expected negative change rate, got %f", resp.ChangeRate30yr)
	}
	if resp.VacancyRateDelta <= 0 {
		t.Errorf("expected positive vacancy delta, got %f", resp.VacancyRateDelta)
	}
}

func TestGetPopulationForecast_UpstreamError(t *testing.T) {
	client := &mockMLITClient{
		populationFunc: func(_ context.Context, z, x, y int) ([]domain.PopulationForecastItem, error) {
			return nil, errors.New("upstream error")
		},
	}
	r := newTestRouter(client)
	req := httptest.NewRequest(http.MethodGet, "/api/population-forecast?lat=35.6762&lng=139.6503", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

// ---- GetLandAppraisals ----

func TestGetLandAppraisals_MissingArea(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/land-appraisals?year=2024", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLandAppraisals_InvalidYear(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
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
	r := newTestRouter(client)
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
	r := newTestRouter(client)
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
	r := newTestRouter(client)
	req := httptest.NewRequest(http.MethodGet, "/api/land-appraisals?area=13&year=2024", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestGetLandAppraisals_InvalidDivision(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/land-appraisals?area=13&year=2024&division=99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestInternalKeyMiddleware_NoKeySet(t *testing.T) {
	t.Setenv("APP_INTERNAL_API_KEY", "")
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when key not set, got %d", w.Code)
	}
}

func TestInternalKeyMiddleware_HealthSkipped(t *testing.T) {
	t.Setenv("APP_INTERNAL_API_KEY", "secret")
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health without key, got %d", w.Code)
	}
}

func TestInternalKeyMiddleware_Unauthorized(t *testing.T) {
	t.Setenv("APP_INTERNAL_API_KEY", "secret")
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/municipalities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without X-Internal-Key, got %d", w.Code)
	}
}

func TestInternalKeyMiddleware_WrongKey(t *testing.T) {
	t.Setenv("APP_INTERNAL_API_KEY", "secret")
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/municipalities", nil)
	req.Header.Set("X-Internal-Key", "wrong")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong key, got %d", w.Code)
	}
}

func TestInternalKeyMiddleware_CorrectKey(t *testing.T) {
	t.Setenv("APP_INTERNAL_API_KEY", "secret")
	r := newTestRouter(&mockMLITClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/municipalities?area=13", nil)
	req.Header.Set("X-Internal-Key", "secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct key, got %d", w.Code)
	}
}

func TestGetUrbanRisks_MissingParams(t *testing.T) {
	r := newTestRouter(&mockMLITClient{})
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
	r := newTestRouter(&mockMLITClient{})
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
	r := newTestRouter(&mockMLITClient{})
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

func TestGetUrbanRisks_WithRisks(t *testing.T) {
	mock := &mockMLITClient{
		embankmentFunc: func(_ context.Context, _, _, _ int) ([]domain.EmbankmentItem, error) {
			return []domain.EmbankmentItem{{Classification: "谷埋め型"}}, nil
		},
		urbanRoadFunc: func(_ context.Context, _, _, _ int) ([]domain.UrbanRoadItem, error) {
			return []domain.UrbanRoadItem{{PlanningRoadJa: "都市計画道路A", KubunID: 3011}}, nil
		},
		disasterHistoryFunc: func(_ context.Context, _, _, _ int) ([]domain.DisasterHistoryItem, error) {
			return []domain.DisasterHistoryItem{{Name: "浸水域", Year: 2019}}, nil
		},
	}
	r := newTestRouter(mock)
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
	codes := make(map[string]bool)
	for _, r := range risks {
		codes[r.Code] = true
	}
	for _, want := range []string{"LARGE_EMBANKMENT", "URBAN_PLANNING_ROAD", "DISASTER_HISTORY"} {
		if !codes[want] {
			t.Errorf("expected risk code %s in response", want)
		}
	}
}

func TestGetUrbanRisks_PartialAPIFailure(t *testing.T) {
	mock := &mockMLITClient{
		embankmentFunc: func(_ context.Context, _, _, _ int) ([]domain.EmbankmentItem, error) {
			return nil, errors.New("API timeout")
		},
		disasterHistoryFunc: func(_ context.Context, _, _, _ int) ([]domain.DisasterHistoryItem, error) {
			return []domain.DisasterHistoryItem{{Name: "がけ崩れ", Year: 2011}}, nil
		},
	}
	r := newTestRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/urban-risks?lat=35.68&lng=139.69", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even with partial failure, got %d", w.Code)
	}
	var risks []domain.UrbanRisk
	if err := json.NewDecoder(w.Body).Decode(&risks); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	found := false
	for _, r := range risks {
		if r.Code == "DISASTER_HISTORY" {
			found = true
		}
	}
	if !found {
		t.Error("expected DISASTER_HISTORY in response despite embankment API failure")
	}
}
