package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// mockMLITClient は service.MLITClient のテスト用モック
type mockMLITClient struct {
	populationFunc   func(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error)
	ridershipFunc    func(ctx context.Context, z, x, y int) ([]mlit.StationRidership, error)
	locationOptFunc  func(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error)
	embankmentFunc   func(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error)
	disasterFunc     func(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error)
	urbanZoningFunc  func(ctx context.Context, z, x, y int) ([]domain.UrbanZoningItem, error)
	liquefactionFunc func(ctx context.Context, z, x, y int) ([]domain.LiquefactionRiskItem, error)
	floodFunc        func(ctx context.Context, z, x, y int) ([]domain.FloodHazardItem, error)
	stormFunc        func(ctx context.Context, z, x, y int) ([]domain.StormHazardItem, error)
	tsunamiFunc      func(ctx context.Context, z, x, y int) ([]domain.TsunamiHazardItem, error)
	landslideFunc    func(ctx context.Context, z, x, y int) ([]domain.LandslideHazardItem, error)
	landPricesFunc   func(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
}

func (m *mockMLITClient) FetchPopulationForecast(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error) {
	if m.populationFunc != nil {
		return m.populationFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchStationRidership(ctx context.Context, z, x, y int) ([]mlit.StationRidership, error) {
	if m.ridershipFunc != nil {
		return m.ridershipFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchLocationOptimization(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error) {
	if m.locationOptFunc != nil {
		return m.locationOptFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchEmbankment(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error) {
	if m.embankmentFunc != nil {
		return m.embankmentFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchDisasterHistory(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error) {
	if m.disasterFunc != nil {
		return m.disasterFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchUrbanZoning(ctx context.Context, z, x, y int) ([]domain.UrbanZoningItem, error) {
	if m.urbanZoningFunc != nil {
		return m.urbanZoningFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchLiquefaction(ctx context.Context, z, x, y int) ([]domain.LiquefactionRiskItem, error) {
	if m.liquefactionFunc != nil {
		return m.liquefactionFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchFloodHazard(ctx context.Context, z, x, y int) ([]domain.FloodHazardItem, error) {
	if m.floodFunc != nil {
		return m.floodFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchStormHazard(ctx context.Context, z, x, y int) ([]domain.StormHazardItem, error) {
	if m.stormFunc != nil {
		return m.stormFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchTsunamiHazard(ctx context.Context, z, x, y int) ([]domain.TsunamiHazardItem, error) {
	if m.tsunamiFunc != nil {
		return m.tsunamiFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchLandslideHazard(ctx context.Context, z, x, y int) ([]domain.LandslideHazardItem, error) {
	if m.landslideFunc != nil {
		return m.landslideFunc(ctx, z, x, y)
	}
	return nil, nil
}
func (m *mockMLITClient) FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
	if m.landPricesFunc != nil {
		return m.landPricesFunc(ctx, q)
	}
	return nil, nil
}

func TestCalcScoreForTile_Success(t *testing.T) {
	mock := &mockMLITClient{
		urbanZoningFunc: func(_ context.Context, _, _, _ int) ([]domain.UrbanZoningItem, error) {
			return []domain.UrbanZoningItem{{AreaClassificationJa: "市街化区域"}}, nil
		},
	}
	svc := NewInvestmentScoreService(mock)
	result, err := svc.CalcScoreForTile(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 市街化区域 +10 → base50 + 10 = 60
	if result.TotalScore != 60 {
		t.Errorf("expected score 60, got %d", result.TotalScore)
	}
	if result.Grade == "" {
		t.Error("grade should not be empty")
	}
}

func TestCalcScoreForTile_PartialFailure(t *testing.T) {
	mock := &mockMLITClient{
		floodFunc: func(_ context.Context, _, _, _ int) ([]domain.FloodHazardItem, error) {
			return nil, errors.New("API timeout")
		},
		urbanZoningFunc: func(_ context.Context, _, _, _ int) ([]domain.UrbanZoningItem, error) {
			return []domain.UrbanZoningItem{{AreaClassificationJa: "市街化区域"}}, nil
		},
	}
	svc := NewInvestmentScoreService(mock)
	result, err := svc.CalcScoreForTile(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("partial API failure should not return error, got: %v", err)
	}
	// flood 失敗 → hazard=0、urban +10 → base50 + 10 = 60
	if result.TotalScore != 60 {
		t.Errorf("expected score 60 with flood API failure, got %d", result.TotalScore)
	}
}

func TestCalcScoreForTile_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := NewInvestmentScoreService(&mockMLITClient{})
	_, err := svc.CalcScoreForTile(ctx, 14, 14547, 6451)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestCalcScoreForTile_AllEmpty(t *testing.T) {
	svc := NewInvestmentScoreService(&mockMLITClient{})
	result, err := svc.CalcScoreForTile(context.Background(), 14, 14547, 6451)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalScore != 50 {
		t.Errorf("all-empty input should yield base score 50, got %d", result.TotalScore)
	}
	if result.Grade != "普通" {
		t.Errorf("expected grade 普通, got %q", result.Grade)
	}
}
