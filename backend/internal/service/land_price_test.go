package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// mockLandClient は LandMLITClient のテスト用モック
type mockLandClient struct {
	fetchFunc     func(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
	appraisalFunc func(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error)
}

func (m *mockLandClient) FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, q)
	}
	return nil, nil
}

func (m *mockLandClient) FetchLandAppraisals(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error) {
	if m.appraisalFunc != nil {
		return m.appraisalFunc(ctx, area, city, year, division)
	}
	return nil, nil
}

func testQuery() mlit.LandPriceQuery {
	return mlit.LandPriceQuery{Area: "13", Year: 2024, Quarter: 1, ToYear: 2024, ToQuarter: 4}
}

// ---- Stats ----

func TestLandStats_Success(t *testing.T) {
	client := &mockLandClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{Period: "2024年第1四半期", TradePrice: 10_000_000, Area: 100, PricePerSqm: 100_000, PricePerTsubo: 330_578},
			}, nil
		},
	}
	svc := NewLandPriceAnalysisService(client)

	stats, err := svc.Stats(context.Background(), testQuery())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Count != 1 {
		t.Errorf("expected count=1, got %d", stats.Count)
	}
	if stats.MedianTsubo == 0 {
		t.Error("expected non-zero MedianTsubo")
	}
}

func TestLandStats_FetchError(t *testing.T) {
	client := &mockLandClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return nil, errors.New("upstream error")
		},
	}
	svc := NewLandPriceAnalysisService(client)

	if _, err := svc.Stats(context.Background(), testQuery()); err == nil {
		t.Fatal("expected error on fetch failure, got nil")
	}
}

// ---- Compare ----

func TestLandCompare_Success(t *testing.T) {
	client := &mockLandClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{Period: "2024年第1四半期", TradePrice: 10_000_000, Area: 100, PricePerSqm: 100_000, PricePerTsubo: 330_578},
			}, nil
		},
	}
	svc := NewLandPriceAnalysisService(client)

	comparison, err := svc.Compare(context.Background(), testQuery(), 10_000_000, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comparison.Assessment == "" {
		t.Error("expected non-empty assessment")
	}
}

func TestLandCompare_FetchError(t *testing.T) {
	client := &mockLandClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return nil, errors.New("upstream error")
		},
	}
	svc := NewLandPriceAnalysisService(client)

	if _, err := svc.Compare(context.Background(), testQuery(), 10_000_000, 100); err == nil {
		t.Fatal("expected error on fetch failure, got nil")
	}
}

// ---- Estimate ----

func TestLandEstimate_Success(t *testing.T) {
	client := &mockLandClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{PricePerTsubo: 200_000, BuildingYear: 2010, StationMinutes: 10},
			}, nil
		},
	}
	svc := NewLandPriceAnalysisService(client)

	result, err := svc.Estimate(context.Background(), testQuery(), domain.TheoreticalPriceInput{
		ListingPrice: 5_000_000,
		LandArea:     100,
		BuildingAge:  10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TheoreticalPriceJPY <= 0 {
		t.Errorf("expected positive theoretical price, got %f", result.TheoreticalPriceJPY)
	}
}

// 取引事例に建築年データがない場合は ErrEstimateDataInsufficient
func TestLandEstimate_DataInsufficient(t *testing.T) {
	client := &mockLandClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{
				{PricePerTsubo: 200_000}, // BuildingYear なし
			}, nil
		},
	}
	svc := NewLandPriceAnalysisService(client)

	_, err := svc.Estimate(context.Background(), testQuery(), domain.TheoreticalPriceInput{
		ListingPrice: 5_000_000,
		LandArea:     100,
	})
	if !errors.Is(err, ErrEstimateDataInsufficient) {
		t.Errorf("expected ErrEstimateDataInsufficient, got %v", err)
	}
}

// ---- Appraisals ----

func TestLandAppraisals_Success(t *testing.T) {
	client := &mockLandClient{
		appraisalFunc: func(_ context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error) {
			return []domain.LandAppraisalItem{
				{Year: 2024, PricePerSqm: 1_000_000, ChangeRate: 0.03, District: "千代田"},
				{Year: 2024, PricePerSqm: 800_000, ChangeRate: 0.02, District: "中央"},
			}, nil
		},
	}
	svc := NewLandPriceAnalysisService(client)

	result, err := svc.Appraisals(context.Background(), "13", "", 2024, "00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AppraisalCount != 2 {
		t.Errorf("AppraisalCount = %d, want 2", result.AppraisalCount)
	}
	if result.AppraisalMedianPerSqm != 900_000 {
		t.Errorf("AppraisalMedianPerSqm = %v, want 900000", result.AppraisalMedianPerSqm)
	}
}

func TestLandAppraisals_NoData(t *testing.T) {
	client := &mockLandClient{
		appraisalFunc: func(_ context.Context, _, _ string, _ int, _ string) ([]domain.LandAppraisalItem, error) {
			return []domain.LandAppraisalItem{}, nil
		},
	}
	svc := NewLandPriceAnalysisService(client)

	_, err := svc.Appraisals(context.Background(), "13", "", 2024, "00")
	if !errors.Is(err, ErrNoAppraisalData) {
		t.Errorf("expected ErrNoAppraisalData, got %v", err)
	}
}

func TestLandAppraisals_FetchError(t *testing.T) {
	client := &mockLandClient{
		appraisalFunc: func(_ context.Context, _, _ string, _ int, _ string) ([]domain.LandAppraisalItem, error) {
			return nil, errors.New("upstream error")
		},
	}
	svc := NewLandPriceAnalysisService(client)

	_, err := svc.Appraisals(context.Background(), "13", "", 2024, "00")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrNoAppraisalData) {
		t.Error("fetch error must not be ErrNoAppraisalData")
	}
}
