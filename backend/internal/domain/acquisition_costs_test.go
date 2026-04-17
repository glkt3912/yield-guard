package domain

import (
	"math"
	"testing"
)

func TestCalcBrokerageFee(t *testing.T) {
	tests := []struct {
		name       string
		price      float64
		multiplier float64
		want       float64 // 円（10円単位で丸め許容）
	}{
		{
			name:       "400万超の標準物件（5900万円）",
			price:      59_000_000,
			multiplier: 1.0,
			// base = 59,000,000*0.03 + 60,000 = 1,830,000; × 1.1 = 2,013,000
			want: 2_013_000,
		},
		{
			name:       "400万超の標準物件（1500万円）",
			price:      15_000_000,
			multiplier: 1.0,
			// base = 15,000,000*0.03 + 60,000 = 510,000; × 1.1 = 561,000
			want: 561_000,
		},
		{
			name:       "仲介手数料無料",
			price:      20_000_000,
			multiplier: 0.0,
			want:       0,
		},
		{
			name:       "半額交渉成立",
			price:      20_000_000,
			multiplier: 0.5,
			// base = 660,000; × 1.1 × 0.5 = 363,000
			want: 363_000,
		},
		{
			name:       "200万円以下",
			price:      1_500_000,
			multiplier: 1.0,
			// base = 1,500,000 * 0.05 = 75,000; × 1.1 = 82,500
			want: 82_500,
		},
		{
			name:       "200万超400万以下（300万）",
			price:      3_000_000,
			multiplier: 1.0,
			// base = 3,000,000*0.04 + 20,000 = 140,000; × 1.1 = 154,000
			want: 154_000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcBrokerageFee(tt.price, tt.multiplier)
			if math.Abs(got-tt.want) > 1 {
				t.Errorf("CalcBrokerageFee(%.0f, %.1f) = %.0f, want %.0f", tt.price, tt.multiplier, got, tt.want)
			}
		})
	}
}

func TestCalcStampDuty(t *testing.T) {
	tests := []struct {
		price float64
		want  float64
	}{
		{500_000, 400},
		{5_000_000, 2_000},
		{10_000_000, 10_000},
		{15_000_000, 20_000},
		{50_000_000, 20_000},
		{59_000_000, 60_000}, // 5000万超1億以下
		{100_000_000, 60_000},
		{200_000_000, 100_000},
	}
	for _, tt := range tests {
		got := CalcStampDuty(tt.price)
		if got != tt.want {
			t.Errorf("CalcStampDuty(%.0f) = %.0f, want %.0f", tt.price, got, tt.want)
		}
	}
}

func TestCalcAcquisitionCosts(t *testing.T) {
	opts := DefaultAcquisitionCostOptions()
	result := CalcAcquisitionCosts(59_000_000, opts)

	if result.BrokerageFee != 2_013_000 {
		t.Errorf("BrokerageFee = %.0f, want 2013000", result.BrokerageFee)
	}
	if result.StampDuty != 60_000 {
		t.Errorf("StampDuty = %.0f, want 60000", result.StampDuty)
	}
	wantTotal := result.BrokerageFee + result.StampDuty
	if result.Total != wantTotal {
		t.Errorf("Total = %.0f, want %.0f", result.Total, wantTotal)
	}
}
