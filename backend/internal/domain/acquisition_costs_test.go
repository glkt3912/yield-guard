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

func TestCalcRegistrationTax(t *testing.T) {
	tests := []struct {
		name             string
		landAssessed     float64
		buildingAssessed float64
		loanAmount       float64
		isNew            bool
		want             float64
	}{
		{
			name:             "中古物件・融資あり",
			landAssessed:     20_000_000, // 土地評価額
			buildingAssessed: 10_000_000, // 建物評価額
			loanAmount:       25_000_000,
			isNew:            false,
			// 土地: 20,000,000×0.02=400,000
			// 建物(中古): 10,000,000×0.02=200,000
			// 抵当権: 25,000,000×0.004=100,000
			want: 700_000,
		},
		{
			name:             "新築物件・融資あり",
			landAssessed:     20_000_000,
			buildingAssessed: 10_000_000,
			loanAmount:       25_000_000,
			isNew:            true,
			// 土地: 400,000
			// 建物(新築): 10,000,000×0.0015=15,000
			// 抵当権: 100,000
			want: 515_000,
		},
		{
			name:             "融資なし",
			landAssessed:     10_000_000,
			buildingAssessed: 5_000_000,
			loanAmount:       0,
			isNew:            false,
			// 土地: 200,000 + 建物: 100,000
			want: 300_000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcRegistrationTax(tt.landAssessed, tt.buildingAssessed, tt.loanAmount, tt.isNew)
			if math.Abs(got-tt.want) > 1 {
				t.Errorf("CalcRegistrationTax = %.0f, want %.0f", got, tt.want)
			}
		})
	}
}

func TestCalcRealEstateAcquisitionTax(t *testing.T) {
	tests := []struct {
		name             string
		landAssessed     float64
		buildingAssessed float64
		want             float64
	}{
		{
			name:             "標準物件",
			landAssessed:     20_000_000,
			buildingAssessed: 10_000_000,
			// 土地: 20,000,000×0.5×0.03=300,000
			// 建物: 10,000,000×0.03=300,000
			want: 600_000,
		},
		{
			name:             "土地のみ",
			landAssessed:     10_000_000,
			buildingAssessed: 0,
			// 土地: 10,000,000×0.5×0.03=150,000
			want: 150_000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcRealEstateAcquisitionTax(tt.landAssessed, tt.buildingAssessed)
			if math.Abs(got-tt.want) > 1 {
				t.Errorf("CalcRealEstateAcquisitionTax = %.0f, want %.0f", got, tt.want)
			}
		})
	}
}

func TestCalcAcquisitionCosts(t *testing.T) {
	// 5900万円の物件: 土地3500万 + 建物2400万, ローン3000万, 中古
	landPrice := 35_000_000.0
	buildingCost := 24_000_000.0
	opts := AcquisitionCostOptions{
		BrokerageMultiplier: 1.0,
		LoanAmount:          30_000_000,
	}
	result := CalcAcquisitionCosts(landPrice, buildingCost, opts)

	// 仲介手数料: (59,000,000×0.03+60,000)×1.1 = 2,013,000
	if result.BrokerageFee != 2_013_000 {
		t.Errorf("BrokerageFee = %.0f, want 2013000", result.BrokerageFee)
	}
	// 印紙税: 60,000
	if result.StampDuty != 60_000 {
		t.Errorf("StampDuty = %.0f, want 60000", result.StampDuty)
	}
	// 登録免許税: 推定モード 土地35,000,000×0.7=24,500,000, 建物24,000,000×0.6=14,400,000
	// 土地: 24,500,000×0.02=490,000, 建物(中古): 14,400,000×0.02=288,000
	// 抵当権: 30,000,000×0.004=120,000 → 合計: 898,000
	if result.RegistrationTax != 898_000 {
		t.Errorf("RegistrationTax = %.0f, want 898000", result.RegistrationTax)
	}
	// 不動産取得税: 土地24,500,000×0.5×0.03=367,500, 建物14,400,000×0.03=432,000 → 合計799,500
	if math.Abs(result.RealEstateAcquisitionTax-799_500) > 1 {
		t.Errorf("RealEstateAcquisitionTax = %.0f, want 799500", result.RealEstateAcquisitionTax)
	}
	wantTotal := result.BrokerageFee + result.StampDuty + result.RegistrationTax + result.RealEstateAcquisitionTax
	if result.Total != wantTotal {
		t.Errorf("Total = %.0f, want %.0f", result.Total, wantTotal)
	}
}
