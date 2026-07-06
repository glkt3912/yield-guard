package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yield-guard/backend/internal/domain"
)

func validInvestmentInput() domain.InvestmentInput {
	input := domain.InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MonthlyRent:     100_000,
		VacancyRate:     0.05,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		MiscExpenseRate: 0.07,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		ExitYieldTarget: 0.06,
		HoldingYears:    10,
	}
	input.Defaults()
	return input
}

// ---- Analyze ----

func TestInvestmentAnalyze_AttachesAISummary(t *testing.T) {
	svc := NewInvestmentAnalysisService(&mockLandClient{}, &stubSummarizer{summary: "AI投資要約"})

	result := svc.Analyze(context.Background(), validInvestmentInput())
	if result.TotalInvestment <= 0 {
		t.Error("expected totalInvestment > 0")
	}
	if result.AISummary != "AI投資要約" {
		t.Errorf("expected AISummary from summarizer, got %q", result.AISummary)
	}
}

func TestInvestmentAnalyze_EmptySummary(t *testing.T) {
	svc := NewInvestmentAnalysisService(&mockLandClient{}, &stubSummarizer{})

	result := svc.Analyze(context.Background(), validInvestmentInput())
	if result.AISummary != "" {
		t.Errorf("expected empty AISummary, got %q", result.AISummary)
	}
}

// ---- RentDeclineHint ----

func TestRentDeclineHint_AllYearsError(t *testing.T) {
	client := &mockLandClient{
		appraisalFunc: func(_ context.Context, _, _ string, _ int, _ string) ([]domain.LandAppraisalItem, error) {
			return nil, errors.New("upstream error")
		},
	}
	svc := NewInvestmentAnalysisService(client, &stubSummarizer{})

	if _, err := svc.RentDeclineHint(context.Background(), "13", ""); err == nil {
		t.Fatal("expected error when all years fail, got nil")
	}
}

// 一部年の失敗は許容し、有効データからヒントを構築する
func TestRentDeclineHint_PartialYearError(t *testing.T) {
	client := &mockLandClient{
		appraisalFunc: func(_ context.Context, _, _ string, year int, _ string) ([]domain.LandAppraisalItem, error) {
			switch year {
			case 2022, 2023:
				return nil, errors.New("upstream error")
			case 2024, 2025, 2026:
				items := make([]domain.LandAppraisalItem, 5)
				base := 180000 - (year-2024)*10000
				for i := range items {
					items[i] = domain.LandAppraisalItem{Year: year, PricePerSqm: float64(base + i*1000), ChangeRate: -0.03}
				}
				return items, nil
			default:
				return []domain.LandAppraisalItem{}, nil
			}
		},
	}
	svc := NewInvestmentAnalysisService(client, &stubSummarizer{})

	hint, err := svc.RentDeclineHint(context.Background(), "13", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hint.Basis != "land_appraisal" {
		t.Errorf("expected basis=land_appraisal, got %q", hint.Basis)
	}
}

func TestRentDeclineHint_DeclineTrend(t *testing.T) {
	// 2022と2024の2年分、各年5件ずつ地価下落傾向のデータを返す
	client := &mockLandClient{
		appraisalFunc: func(_ context.Context, _, _ string, year int, _ string) ([]domain.LandAppraisalItem, error) {
			switch year {
			case 2022:
				return []domain.LandAppraisalItem{
					{Year: 2022, PricePerSqm: 200000, ChangeRate: -0.02},
					{Year: 2022, PricePerSqm: 210000, ChangeRate: -0.02},
					{Year: 2022, PricePerSqm: 190000, ChangeRate: -0.02},
					{Year: 2022, PricePerSqm: 205000, ChangeRate: -0.02},
					{Year: 2022, PricePerSqm: 195000, ChangeRate: -0.02},
				}, nil
			case 2024:
				return []domain.LandAppraisalItem{
					{Year: 2024, PricePerSqm: 180000, ChangeRate: -0.05},
					{Year: 2024, PricePerSqm: 185000, ChangeRate: -0.05},
					{Year: 2024, PricePerSqm: 175000, ChangeRate: -0.05},
					{Year: 2024, PricePerSqm: 182000, ChangeRate: -0.05},
					{Year: 2024, PricePerSqm: 178000, ChangeRate: -0.05},
				}, nil
			default:
				return []domain.LandAppraisalItem{}, nil
			}
		},
	}
	svc := NewInvestmentAnalysisService(client, &stubSummarizer{})

	hint, err := svc.RentDeclineHint(context.Background(), "13", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hint.Basis != "land_appraisal" {
		t.Errorf("expected basis=land_appraisal, got %q", hint.Basis)
	}
	if hint.HintRate <= 0 {
		t.Errorf("expected hintRate > 0 for declining land prices, got %f", hint.HintRate)
	}
	if hint.FallbackUsed {
		t.Error("expected fallbackUsed=false for land_appraisal basis")
	}
}

func TestRentDeclineHint_RisingTrend(t *testing.T) {
	// 地価上昇傾向 → fallback を返す
	client := &mockLandClient{
		appraisalFunc: func(_ context.Context, _, _ string, year int, _ string) ([]domain.LandAppraisalItem, error) {
			switch year {
			case 2022:
				return []domain.LandAppraisalItem{
					{Year: 2022, PricePerSqm: 100000, ChangeRate: 0.05},
					{Year: 2022, PricePerSqm: 100000, ChangeRate: 0.05},
					{Year: 2022, PricePerSqm: 100000, ChangeRate: 0.05},
					{Year: 2022, PricePerSqm: 100000, ChangeRate: 0.05},
					{Year: 2022, PricePerSqm: 100000, ChangeRate: 0.05},
				}, nil
			case 2024:
				return []domain.LandAppraisalItem{
					{Year: 2024, PricePerSqm: 120000, ChangeRate: 0.05},
					{Year: 2024, PricePerSqm: 120000, ChangeRate: 0.05},
					{Year: 2024, PricePerSqm: 120000, ChangeRate: 0.05},
					{Year: 2024, PricePerSqm: 120000, ChangeRate: 0.05},
					{Year: 2024, PricePerSqm: 120000, ChangeRate: 0.05},
				}, nil
			default:
				return []domain.LandAppraisalItem{}, nil
			}
		},
	}
	svc := NewInvestmentAnalysisService(client, &stubSummarizer{})

	hint, err := svc.RentDeclineHint(context.Background(), "13", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hint.Basis != "fallback" {
		t.Errorf("expected basis=fallback for rising prices, got %q", hint.Basis)
	}
	if !hint.FallbackUsed {
		t.Error("expected fallbackUsed=true for rising prices")
	}
}

func TestRentDeclineHint_InsufficientData(t *testing.T) {
	// データが5件未満 → fallback
	client := &mockLandClient{
		appraisalFunc: func(_ context.Context, _, _ string, year int, _ string) ([]domain.LandAppraisalItem, error) {
			if year == 2024 {
				// 4件だけ返す（minDataPointsForHint=5未満）
				return []domain.LandAppraisalItem{
					{Year: 2024, PricePerSqm: 150000, ChangeRate: -0.01},
					{Year: 2024, PricePerSqm: 160000, ChangeRate: -0.01},
					{Year: 2024, PricePerSqm: 155000, ChangeRate: -0.01},
					{Year: 2024, PricePerSqm: 145000, ChangeRate: -0.01},
				}, nil
			}
			return []domain.LandAppraisalItem{}, nil
		},
	}
	svc := NewInvestmentAnalysisService(client, &stubSummarizer{})

	hint, err := svc.RentDeclineHint(context.Background(), "13", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hint.Basis != "fallback" {
		t.Errorf("expected basis=fallback for insufficient data, got %q", hint.Basis)
	}
}
