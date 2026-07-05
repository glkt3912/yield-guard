package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// mockRentClient は RentMLITClient のテスト用モック
type mockRentClient struct {
	rentStatsFunc func(ctx context.Context, q mlit.LandPriceQuery, areaSqm float64) (domain.RentStatsResult, error)
}

func (m *mockRentClient) FetchRentStats(ctx context.Context, q mlit.LandPriceQuery, areaSqm float64) (domain.RentStatsResult, error) {
	if m.rentStatsFunc != nil {
		return m.rentStatsFunc(ctx, q, areaSqm)
	}
	return domain.RentStatsResult{}, nil
}

func TestRentStats_Success(t *testing.T) {
	client := &mockRentClient{
		rentStatsFunc: func(_ context.Context, _ mlit.LandPriceQuery, _ float64) (domain.RentStatsResult, error) {
			return domain.RentStatsResult{Median: 80000, Average: 82000, Count: 15}, nil
		},
	}
	svc := NewRentStatsService(client)

	result := svc.Stats(context.Background(), "13", "", 0)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Median != 80000 || result.Average != 82000 || result.Count != 15 {
		t.Errorf("unexpected result: %+v", result)
	}
	if result.LowConfidence {
		t.Error("expected LowConfidence=false for count=15")
	}
}

// サンプル3件未満は低信頼フラグを立てる
func TestRentStats_LowConfidence(t *testing.T) {
	client := &mockRentClient{
		rentStatsFunc: func(_ context.Context, _ mlit.LandPriceQuery, _ float64) (domain.RentStatsResult, error) {
			return domain.RentStatsResult{Median: 80000, Count: 2}, nil
		},
	}
	svc := NewRentStatsService(client)

	result := svc.Stats(context.Background(), "13", "", 0)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.LowConfidence {
		t.Error("expected LowConfidence=true for count=2")
	}
}

func TestRentStats_NoData(t *testing.T) {
	svc := NewRentStatsService(&mockRentClient{})

	if result := svc.Stats(context.Background(), "13", "", 0); result != nil {
		t.Errorf("expected nil for count=0, got %+v", result)
	}
}

func TestRentStats_FetchError(t *testing.T) {
	client := &mockRentClient{
		rentStatsFunc: func(_ context.Context, _ mlit.LandPriceQuery, _ float64) (domain.RentStatsResult, error) {
			return domain.RentStatsResult{}, errors.New("upstream error")
		},
	}
	svc := NewRentStatsService(client)

	// エラーはサイレントに nil を返す
	if result := svc.Stats(context.Background(), "13", "", 0); result != nil {
		t.Errorf("expected nil on fetch error, got %+v", result)
	}
}

// 直近2年 + 直前四半期までのクエリが組み立てられること
func TestRentStats_QueryRange(t *testing.T) {
	var got mlit.LandPriceQuery
	client := &mockRentClient{
		rentStatsFunc: func(_ context.Context, q mlit.LandPriceQuery, _ float64) (domain.RentStatsResult, error) {
			got = q
			return domain.RentStatsResult{Count: 5}, nil
		},
	}
	svc := NewRentStatsService(client)
	svc.Stats(context.Background(), "13", "13101", 25)

	now := time.Now()
	wantToYear := now.Year()
	wantToQuarter := (int(now.Month()) - 1) / 3
	if wantToQuarter == 0 {
		wantToQuarter = 4
		wantToYear--
	}

	if got.Area != "13" || got.City != "13101" {
		t.Errorf("unexpected area/city: %+v", got)
	}
	if got.Quarter != 1 {
		t.Errorf("expected from-quarter=1, got %d", got.Quarter)
	}
	if got.ToYear != wantToYear || got.ToQuarter != wantToQuarter {
		t.Errorf("expected to=%d Q%d, got to=%d Q%d", wantToYear, wantToQuarter, got.ToYear, got.ToQuarter)
	}
	if got.Year != wantToYear-2 {
		t.Errorf("expected from-year=%d, got %d", wantToYear-2, got.Year)
	}
}
