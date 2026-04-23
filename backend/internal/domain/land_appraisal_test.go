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

func TestCalcRentDeclineHint_FallbackWhenTooFewPoints(t *testing.T) {
	// 総件数 < 5 → fallback
	itemsByYear := map[int][]LandAppraisalItem{
		2022: {{Year: 2022, PricePerSqm: 100_000}},
		2024: {{Year: 2024, PricePerSqm: 90_000}},
	}
	got := CalcRentDeclineHint(itemsByYear)
	if !got.FallbackUsed {
		t.Errorf("expected fallback when dataPointCount < 5, got %+v", got)
	}
	if got.DataPointCount != 2 {
		t.Errorf("expected dataPointCount=2, got %v", got.DataPointCount)
	}
}

func TestCalcRentDeclineHint_FallbackWhenOnlyOneYear(t *testing.T) {
	// 有効年数 < 2 → fallback
	items := make([]LandAppraisalItem, 10)
	for i := range items {
		items[i] = LandAppraisalItem{Year: 2024, PricePerSqm: 100_000}
	}
	got := CalcRentDeclineHint(map[int][]LandAppraisalItem{2024: items})
	if !got.FallbackUsed {
		t.Errorf("expected fallback when only one year available, got %+v", got)
	}
}

func TestCalcRentDeclineHint_DeclineTrend(t *testing.T) {
	// 地価が下落（5点ずつ、2年間で100000→90000）
	makeItems := func(year int, price float64, n int) []LandAppraisalItem {
		items := make([]LandAppraisalItem, n)
		for i := range items {
			items[i] = LandAppraisalItem{Year: year, PricePerSqm: price}
		}
		return items
	}
	itemsByYear := map[int][]LandAppraisalItem{
		2022: makeItems(2022, 100_000, 5),
		2024: makeItems(2024, 90_000, 5),
	}
	got := CalcRentDeclineHint(itemsByYear)
	if got.FallbackUsed {
		t.Errorf("expected no fallback for declining trend, got %+v", got)
	}
	if got.Basis != "land_appraisal" {
		t.Errorf("expected basis=land_appraisal, got %q", got.Basis)
	}
	if got.HintRate <= 0 {
		t.Errorf("expected positive hintRate, got %v", got.HintRate)
	}
}

func TestCalcRentDeclineHint_RiseTrend(t *testing.T) {
	// 地価が上昇 → fallback
	makeItems := func(year int, price float64, n int) []LandAppraisalItem {
		items := make([]LandAppraisalItem, n)
		for i := range items {
			items[i] = LandAppraisalItem{Year: year, PricePerSqm: price}
		}
		return items
	}
	itemsByYear := map[int][]LandAppraisalItem{
		2022: makeItems(2022, 100_000, 5),
		2024: makeItems(2024, 120_000, 5),
	}
	got := CalcRentDeclineHint(itemsByYear)
	if !got.FallbackUsed {
		t.Errorf("expected fallback for rising trend, got %+v", got)
	}
	if got.HintRate != 0 {
		t.Errorf("expected hintRate=0 for rising trend, got %v", got.HintRate)
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
