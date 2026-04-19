package domain

import (
	"testing"
)

func TestCalcAppraisalComparison_Empty(t *testing.T) {
	result := CalcAppraisalComparison(nil)
	if result.AppraisalMedianPerSqm != 0 || result.AppraisalCount != 0 {
		t.Errorf("expected zero result for empty input, got %+v", result)
	}
}

func TestCalcAppraisalComparison_Median(t *testing.T) {
	items := []LandAppraisalItem{
		{Year: 2024, PricePerSqm: 100_000, ChangeRate: 0.02},
		{Year: 2024, PricePerSqm: 200_000, ChangeRate: 0.05},
		{Year: 2024, PricePerSqm: 300_000, ChangeRate: 0.08},
	}
	result := CalcAppraisalComparison(items)
	if result.AppraisalMedianPerSqm != 200_000 {
		t.Errorf("expected median 200000, got %v", result.AppraisalMedianPerSqm)
	}
	if result.AppraisalCount != 3 {
		t.Errorf("expected count 3, got %v", result.AppraisalCount)
	}
}

func TestCalcAppraisalComparison_MedianEven(t *testing.T) {
	items := []LandAppraisalItem{
		{Year: 2024, PricePerSqm: 100_000, ChangeRate: 0.0},
		{Year: 2024, PricePerSqm: 200_000, ChangeRate: 0.0},
	}
	result := CalcAppraisalComparison(items)
	if result.AppraisalMedianPerSqm != 150_000 {
		t.Errorf("expected median 150000, got %v", result.AppraisalMedianPerSqm)
	}
}

func TestCalcAppraisalComparison_TrendLabel(t *testing.T) {
	tests := []struct {
		trend float64
		want  string
	}{
		{0.05, "上昇"},
		{0.031, "上昇"},
		{0.03, "安定"}, // boundary: > 0.03 is 上昇, so exactly 0.03 is 安定
		{0.02, "安定"},
		{0.0, "安定"},
		{-0.03, "安定"}, // boundary: < -0.03 is 下落, so exactly -0.03 is 安定
		{-0.031, "下落"},
	}
	for _, tt := range tests {
		got := appraisalTrendLabel(tt.trend)
		if got != tt.want {
			t.Errorf("appraisalTrendLabel(%v) = %q, want %q", tt.trend, got, tt.want)
		}
	}
}

func TestCalcAppraisalComparison_SkipsZeroPrice(t *testing.T) {
	items := []LandAppraisalItem{
		{Year: 2024, PricePerSqm: 0, ChangeRate: 0.0},
		{Year: 2024, PricePerSqm: 500_000, ChangeRate: 0.04},
	}
	result := CalcAppraisalComparison(items)
	if result.AppraisalMedianPerSqm != 500_000 {
		t.Errorf("expected median 500000 (skipping zero price), got %v", result.AppraisalMedianPerSqm)
	}
	if result.AppraisalCount != 2 {
		t.Errorf("expected count 2 (all items counted), got %v", result.AppraisalCount)
	}
}
