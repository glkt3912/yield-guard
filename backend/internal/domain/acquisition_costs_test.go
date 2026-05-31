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
			// 土地: 20,000,000×0.02=400,000
			// 建物(新築保存登記・本則): 10,000,000×0.004=40,000
			// 抵当権: 25,000,000×0.004=100,000
			want: 540_000,
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

// TestCalcBrokerageFee_BoundaryValues は宅建業法46条の3段階境界値を網羅する
func TestCalcBrokerageFee_BoundaryValues(t *testing.T) {
	tests := []struct {
		name  string
		price float64
		want  float64
	}{
		{
			name:  "ちょうど200万円（200万以下の上限境界）",
			price: 2_000_000,
			// base = 2,000,000 * 0.05 = 100,000; × 1.1 = 110,000
			want: 110_000,
		},
		{
			name:  "200万1円（200万超400万以下の下限境界）",
			price: 2_000_001,
			// base = 2,000,001*0.04 + 20,000 = 100,000.04; × 1.1 ≈ 110,000
			want: 110_000,
		},
		{
			name:  "ちょうど400万円（200万超400万以下の上限境界）",
			price: 4_000_000,
			// base = 4,000,000*0.04 + 20,000 = 180,000; × 1.1 = 198,000
			want: 198_000,
		},
		{
			name:  "400万1円（400万超の下限境界）",
			price: 4_000_001,
			// base = 4,000,001*0.03 + 60,000 = 180,000.03; × 1.1 ≈ 198,000
			want: 198_000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcBrokerageFee(tt.price, 1.0)
			if math.Abs(got-tt.want) > 10 {
				t.Errorf("CalcBrokerageFee(%.0f) = %.0f, want %.0f", tt.price, got, tt.want)
			}
		})
	}
}

// TestCalcStampDuty_BoundaryValues は印紙税の各区分境界値を網羅する
func TestCalcStampDuty_BoundaryValues(t *testing.T) {
	tests := []struct {
		price float64
		want  float64
	}{
		{100_000, 200},        // ちょうど10万円（最低区分上限）
		{100_001, 400},        // 10万1円（5千万以下区分の下限）
		{500_000, 400},        // ちょうど50万円
		{500_001, 1_000},      // 50万1円（100万以下区分）
		{1_000_000, 1_000},    // ちょうど100万円
		{1_000_001, 2_000},    // 100万1円（500万以下区分）
		{5_000_000, 2_000},    // ちょうど500万円
		{5_000_001, 10_000},   // 500万1円（1千万以下区分）
		{10_000_000, 10_000},  // ちょうど1千万円
		{10_000_001, 20_000},  // 1千万1円（5千万以下区分）
		{50_000_000, 20_000},  // ちょうど5千万円
		{50_000_001, 60_000},  // 5千万1円（1億以下区分）
		{100_000_000, 60_000}, // ちょうど1億円
		{100_000_001, 100_000}, // 1億1円（5億以下区分）
	}
	for _, tt := range tests {
		got := CalcStampDuty(tt.price)
		if got != tt.want {
			t.Errorf("CalcStampDuty(%.0f) = %.0f, want %.0f", tt.price, got, tt.want)
		}
	}
}

// TestCalcRegistrationTax_Patterns は登録免許税の4パターンを網羅する
func TestCalcRegistrationTax_Patterns(t *testing.T) {
	tests := []struct {
		name             string
		landAssessed     float64
		buildingAssessed float64
		loanAmount       float64
		isNew            bool
		want             float64
	}{
		{
			name:             "新築・融資なし（土地移転+建物保存）",
			landAssessed:     10_000_000,
			buildingAssessed: 8_000_000,
			loanAmount:       0,
			isNew:            true,
			// 土地: 10,000,000×0.02=200,000
			// 建物(新築保存登記・本則): 8,000,000×0.004=32,000
			// 抵当権: 0
			want: 232_000,
		},
		{
			name:             "中古・融資なし（土地移転+建物移転）",
			landAssessed:     10_000_000,
			buildingAssessed: 8_000_000,
			loanAmount:       0,
			isNew:            false,
			// 土地: 200,000 + 建物: 160,000 = 360,000
			want: 360_000,
		},
		{
			name:             "土地のみ移転（建物なし）",
			landAssessed:     15_000_000,
			buildingAssessed: 0,
			loanAmount:       0,
			isNew:            false,
			// 土地: 15,000,000×0.02=300,000
			want: 300_000,
		},
		{
			name:             "抵当権設定のみ（土地建物ゼロ）",
			landAssessed:     0,
			buildingAssessed: 0,
			loanAmount:       20_000_000,
			isNew:            false,
			// 抵当権: 20,000,000×0.004=80,000
			want: 80_000,
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

func TestCalcAcquisitionCosts_NewBuilding(t *testing.T) {
	// 5900万円: 土地3500万 + 建物2400万, ローン3000万, 新築
	landPrice := 35_000_000.0
	buildingCost := 24_000_000.0
	opts := AcquisitionCostOptions{
		BrokerageMultiplier: 1.0,
		LoanAmount:          30_000_000,
		IsNewBuilding:       true,
	}
	result := CalcAcquisitionCosts(landPrice, buildingCost, opts)

	// 仲介手数料: 土地のみ基準 (35,000,000×0.03+60,000)×1.1 = 1,221,000
	if result.BrokerageFee != 1_221_000 {
		t.Errorf("BrokerageFee = %.0f, want 1221000", result.BrokerageFee)
	}
	// 登録免許税: 土地移転2% + 建物保存登記(本則)0.4% + 抵当権0.4%
	// 土地推定: 35,000,000×0.7=24,500,000 → 24,500,000×0.02=490,000
	// 建物推定: 24,000,000×0.6=14,400,000 → 14,400,000×0.004=57,600
	// 抵当権: 30,000,000×0.004=120,000 → 合計: 667,600
	if result.RegistrationTax != 667_600 {
		t.Errorf("RegistrationTax = %.0f, want 667600", result.RegistrationTax)
	}
	wantTotal := result.BrokerageFee + result.StampDuty + result.RegistrationTax + result.RealEstateAcquisitionTax
	if result.Total != wantTotal {
		t.Errorf("Total = %.0f, want %.0f", result.Total, wantTotal)
	}
}
