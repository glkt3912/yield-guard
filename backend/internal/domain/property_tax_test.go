package domain

import (
	"math"
	"testing"
)

func TestCalcPropertyTax(t *testing.T) {
	tests := []struct {
		name             string
		assessedLand     float64
		assessedBuilding float64
		opts             PropertyTaxOptions
		wantAnnualTotal  float64
	}{
		{
			name:             "特例なし（面積0）",
			assessedLand:     20_000_000,
			assessedBuilding: 10_000_000,
			opts:             PropertyTaxOptions{},
			// 土地: 20,000,000×0.014=280,000 + 都市計画: 20,000,000×0.003=60,000
			// 建物: 10,000,000×0.014=140,000 + 都市計画: 10,000,000×0.003=30,000
			// 合計: 510,000
			wantAnnualTotal: 510_000,
		},
		{
			name:             "小規模住宅用地特例（150㎡）",
			assessedLand:     20_000_000,
			assessedBuilding: 10_000_000,
			opts:             PropertyTaxOptions{LandAreaSqm: 150},
			// 土地固定: 20,000,000×0.014/6=46,667 → 46,667
			// 土地都市: 20,000,000×0.003/3=20,000
			// 建物固定: 140,000, 建物都市: 30,000
			// 合計: 46,667 + 20,000 + 140,000 + 30,000 = 236,667
			wantAnnualTotal: 236_667,
		},
		{
			name:             "一般住宅用地特例（300㎡）",
			assessedLand:     20_000_000,
			assessedBuilding: 10_000_000,
			opts:             PropertyTaxOptions{LandAreaSqm: 300},
			// 土地固定: 20,000,000×0.014/3=93,333
			// 土地都市: 20,000,000×0.003×2/3=40,000
			// 建物: 170,000
			// 合計: 303,333
			wantAnnualTotal: 303_333,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcPropertyTax(tt.assessedLand, tt.assessedBuilding, tt.opts)
			if math.Abs(got.AnnualTotal-tt.wantAnnualTotal) > 1 {
				t.Errorf("AnnualTotal = %.0f, want %.0f", got.AnnualTotal, tt.wantAnnualTotal)
			}
		})
	}
}

// TestCalcPropertyTax_200SqmBoundary はちょうど200㎡の境界値（小規模1/6 vs 一般1/3）を検証する
func TestCalcPropertyTax_200SqmBoundary(t *testing.T) {
	assessedLand := 20_000_000.0
	assessedBuilding := 10_000_000.0

	// ちょうど200㎡: 小規模住宅用地特例（1/6, 1/3）が適用されるべき
	gotAt200 := CalcPropertyTax(assessedLand, assessedBuilding, PropertyTaxOptions{LandAreaSqm: 200})
	// 土地固定: 20,000,000×0.014/6=46,667
	// 土地都市: 20,000,000×0.003/3=20,000
	// 建物固定: 140,000, 建物都市: 30,000
	wantAt200 := 236_667.0
	if math.Abs(gotAt200.AnnualTotal-wantAt200) > 1 {
		t.Errorf("200㎡(小規模): AnnualTotal = %.0f, want %.0f", gotAt200.AnnualTotal, wantAt200)
	}

	// 200㎡超（201㎡）: 一般住宅用地特例（1/3, 2/3）が適用されるべき
	gotAt201 := CalcPropertyTax(assessedLand, assessedBuilding, PropertyTaxOptions{LandAreaSqm: 201})
	// 土地固定: 20,000,000×0.014/3=93,333
	// 土地都市: 20,000,000×0.003×2/3=40,000
	// 建物: 170,000
	wantAt201 := 303_333.0
	if math.Abs(gotAt201.AnnualTotal-wantAt201) > 1 {
		t.Errorf("201㎡(一般): AnnualTotal = %.0f, want %.0f", gotAt201.AnnualTotal, wantAt201)
	}

	// 200㎡は小規模（低税額）、201㎡は一般（高税額）になること
	if gotAt200.AnnualTotal >= gotAt201.AnnualTotal {
		t.Errorf("200㎡(%.0f) >= 201㎡(%.0f): 小規模特例が一般より税額が低いはず", gotAt200.AnnualTotal, gotAt201.AnnualTotal)
	}

	// 土地の内訳が正しく切り替わること
	// 200㎡: 小規模 landFixed=46,667
	if math.Abs(gotAt200.LandFixedAssetTax-46_667) > 1 {
		t.Errorf("200㎡ LandFixedAssetTax = %.0f, want 46667", gotAt200.LandFixedAssetTax)
	}
	// 201㎡: 一般 landFixed=93,333
	if math.Abs(gotAt201.LandFixedAssetTax-93_333) > 1 {
		t.Errorf("201㎡ LandFixedAssetTax = %.0f, want 93333", gotAt201.LandFixedAssetTax)
	}
}

func TestCalcPropertyTaxProration(t *testing.T) {
	tests := []struct {
		name      string
		annualTax float64
		month     int
		day       int
		year      int
		want      float64
	}{
		{
			name:      "7月1日引渡し・平年（184日分）",
			annualTax: 365_000,
			month:     7,
			day:       1,
			year:      2025, // 平年
			// 通算: 31+28+31+30+31+30+1=182日目
			// 買主負担: 365-182+1=184日
			// 365,000 × 184/365 = 184,000
			want: 184_000,
		},
		{
			name:      "7月1日引渡し・うるう年（184日分）",
			annualTax: 366_000,
			month:     7,
			day:       1,
			year:      2024, // うるう年
			// 通算: 31+29+31+30+31+30+1=183日目
			// 買主負担: 366-183+1=184日
			// 366,000 × 184/366 = 184,000
			want: 184_000,
		},
		{
			name:      "3月1日引渡し・うるう年（うるう日を含む）",
			annualTax: 366_000,
			month:     3,
			day:       1,
			year:      2024, // うるう年: 1月=31, 2月=29
			// 通算: 31+29+1=61日目
			// 買主負担: 366-61+1=306日
			// 366,000 × 306/366 = 306,000
			want: 306_000,
		},
		{
			name:      "1月1日引渡し（全日分）",
			annualTax: 365_000,
			month:     1,
			day:       1,
			year:      2025,
			want:      365_000,
		},
		{
			name:      "12月31日引渡し（1日分）",
			annualTax: 365_000,
			month:     12,
			day:       31,
			year:      2025,
			want:      1_000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcPropertyTaxProration(tt.annualTax, tt.month, tt.day, tt.year)
			if math.Abs(got-tt.want) > 1 {
				t.Errorf("CalcPropertyTaxProration = %.0f, want %.0f", got, tt.want)
			}
		})
	}
}

func TestCalcAcquisitionCostsWithProration(t *testing.T) {
	opts := AcquisitionCostOptions{
		BrokerageMultiplier: 1.0,
		LoanAmount:          30_000_000,
		DeliveryYear:        2025,
		DeliveryMonth:       7,
		DeliveryDay:         1,
		LandAreaSqm:         150,
	}
	result := CalcAcquisitionCosts(35_000_000, 24_000_000, opts)

	if result.PropertyTaxProration <= 0 {
		t.Errorf("PropertyTaxProration should be > 0, got %.0f", result.PropertyTaxProration)
	}
	wantTotal := result.BrokerageFee + result.StampDuty + result.RegistrationTax +
		result.RealEstateAcquisitionTax + result.PropertyTaxProration
	if math.Abs(result.Total-wantTotal) > 1 {
		t.Errorf("Total = %.0f, want %.0f", result.Total, wantTotal)
	}
}
