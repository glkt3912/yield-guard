package domain

import (
	"testing"
)

const taxEpsilon = 1.0 // 1円単位の誤差を許容

func TestCalcIncomeTax(t *testing.T) {
	tests := []struct {
		name        string
		taxable     float64
		wantApprox  float64
	}{
		{"ゼロ", 0, 0},
		{"負の課税所得", -500_000, 0},
		{"195万以下(5%)", 1_000_000, 50_000},
		{"330万以下(10%)", 2_000_000, 102_500},
		{"695万以下(20%)", 5_000_000, 572_500},
		{"900万以下(23%)", 8_000_000, 1_204_000},
		{"1800万以下(33%)", 12_000_000, 2_424_000},
		{"4000万以下(40%)", 20_000_000, 5_204_000},
		{"4000万超(45%)", 50_000_000, 17_704_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcIncomeTax(tt.taxable)
			if !approxEqual(got, tt.wantApprox, taxEpsilon) {
				t.Errorf("calcIncomeTax(%.0f) = %.0f, want %.0f", tt.taxable, got, tt.wantApprox)
			}
		})
	}
}

func TestCalcSalaryDeduction(t *testing.T) {
	tests := []struct {
		name       string
		salary     float64
		wantApprox float64
	}{
		{"162.5万以下", 1_500_000, 550_000},
		{"180万以下", 1_700_000, 580_000},       // 1700000*0.4-100000
		{"360万以下", 3_000_000, 980_000},       // 3000000*0.3+80000
		{"660万以下", 6_000_000, 1_640_000},     // 6000000*0.2+440000
		{"850万以下", 8_000_000, 1_900_000},     // 8000000*0.1+1100000
		{"850万超", 10_000_000, 1_950_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcSalaryDeduction(tt.salary)
			if !approxEqual(got, tt.wantApprox, taxEpsilon) {
				t.Errorf("calcSalaryDeduction(%.0f) = %.0f, want %.0f", tt.salary, got, tt.wantApprox)
			}
		})
	}
}

func TestCalcBasicDeduction(t *testing.T) {
	tests := []struct {
		name        string
		totalIncome float64
		want        float64
	}{
		{"2400万円以下: 上限", 24_000_000, 480_000},
		{"2400万円超: 下限+1", 24_000_001, 320_000},
		{"2450万円以下: 上限", 24_500_000, 320_000},
		{"2450万円超: 下限+1", 24_500_001, 160_000},
		{"2500万円以下: 上限", 25_000_000, 160_000},
		{"2500万円超: ゼロ", 25_000_001, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcBasicDeduction(tt.totalIncome)
			if got != tt.want {
				t.Errorf("calcBasicDeduction(%.0f) = %.0f, want %.0f", tt.totalIncome, got, tt.want)
			}
		})
	}
}

func TestCalcTaxSimulation_NilWhenNoSalary(t *testing.T) {
	input := InvestmentInput{SalaryIncome: 0, HoldingYears: 10}
	result := CalcTaxSimulation(input, []YearlyResult{}, 0)
	if result != nil {
		t.Error("SalaryIncome=0のとき nil を返すべき")
	}
}

func TestCalcTaxSimulation_LossCarryover(t *testing.T) {
	// 年収600万円の投資家が減価償却で不動産所得が赤字の年に節税が発生すること
	salary := 6_000_000.0
	// 課税所得: 6000000 - 1640000(控除) - 480000(基礎) = 3880000
	// 税額(概算): 3880000*0.20 - 427500 + 3880000*0.021 + 3880000*0.10

	yearly := []YearlyResult{
		{Year: 1, TaxableIncome: -500_000}, // 赤字(節税)
		{Year: 2, TaxableIncome: 200_000},  // 黒字(増税)
		{Year: 3, TaxableIncome: -300_000}, // 赤字(節税)
	}

	input := InvestmentInput{
		SalaryIncome: salary,
		HoldingYears: 3,
	}

	result := CalcTaxSimulation(input, yearly, 1_000_000)
	if result == nil {
		t.Fatal("SalaryIncome>0のとき non-nil を返すべき")
	}

	rows := result.SalaryLossCarryover.YearlyRows
	if len(rows) != 3 {
		t.Fatalf("yearlyRows の長さが 3 であるべき、got %d", len(rows))
	}

	// 赤字年は節税（TaxDifference > 0）
	if rows[0].TaxDifference <= 0 {
		t.Errorf("1年目(赤字): 節税額 > 0 を期待、got %.0f", rows[0].TaxDifference)
	}
	// 黒字年は増税（TaxDifference < 0）
	if rows[1].TaxDifference >= 0 {
		t.Errorf("2年目(黒字): 増税額 < 0 を期待、got %.0f", rows[1].TaxDifference)
	}
	// 3年目も赤字なので節税
	if rows[2].TaxDifference <= 0 {
		t.Errorf("3年目(赤字): 節税額 > 0 を期待、got %.0f", rows[2].TaxDifference)
	}
}

func TestCalcTaxSimulation_OwnershipBreakeven(t *testing.T) {
	// 高年収（2000万）で不動産黒字が続く場合、法人が有利になる年が存在すること
	salary := 20_000_000.0 // 課税後は高い累進税率ゾーン

	yearly := make([]YearlyResult, 15)
	for i := range yearly {
		yearly[i] = YearlyResult{Year: i + 1, TaxableIncome: 3_000_000} // 毎年黒字
	}

	input := InvestmentInput{
		SalaryIncome: salary,
		HoldingYears: 15,
	}

	result := CalcTaxSimulation(input, yearly, 5_000_000)
	if result == nil {
		t.Fatal("結果が nil")
	}

	cmp := result.OwnershipComparison
	// 高所得者は法人税率(33.4%) < 個人実効税率(40%超)なので早期に法人が有利
	if cmp.BreakevenYear == -1 {
		t.Error("高年収(2000万)で法人が有利になる年が存在するはず")
	}
	if cmp.Corporate.TotalTaxBurden >= cmp.Individual.TotalTaxBurden {
		t.Errorf("法人合計税負担(%.0f) < 個人(%.0f) であるべき",
			cmp.Corporate.TotalTaxBurden, cmp.Individual.TotalTaxBurden)
	}
}

func TestCalcTaxSimulation_LowIncomeNeverBreakeven(t *testing.T) {
	// 低年収（300万）では個人累進税率が低く、法人化のメリットが保有期間内に出ないこと
	salary := 3_000_000.0

	yearly := make([]YearlyResult, 10)
	for i := range yearly {
		yearly[i] = YearlyResult{Year: i + 1, TaxableIncome: 500_000}
	}

	input := InvestmentInput{
		SalaryIncome: salary,
		HoldingYears: 10,
	}

	result := CalcTaxSimulation(input, yearly, 1_000_000)
	if result == nil {
		t.Fatal("結果が nil")
	}

	cmp := result.OwnershipComparison
	// 低所得者は法人税率(33.4%) > 個人実効税率(15%程度)なので法人不利
	if cmp.BreakevenYear != -1 {
		t.Errorf("低年収(300万)では法人有利年は -1 であるべき、got %d", cmp.BreakevenYear)
	}
}

func TestCalcTaxSimulation_NegativeExitCapitalGain(t *testing.T) {
	// 譲渡損（exitCapitalGain < 0）のとき TransferTax が 0 になること
	yearly := []YearlyResult{
		{Year: 1, TaxableIncome: 100_000},
	}
	input := InvestmentInput{
		SalaryIncome: 6_000_000,
		HoldingYears: 1,
	}

	result := CalcTaxSimulation(input, yearly, -500_000)
	if result == nil {
		t.Fatal("結果が nil")
	}

	indiv := result.OwnershipComparison.Individual
	corp := result.OwnershipComparison.Corporate

	if indiv.TransferTax != 0 {
		t.Errorf("譲渡損のとき個人 TransferTax = 0 を期待、got %.0f", indiv.TransferTax)
	}
	if corp.TransferTax != 0 {
		t.Errorf("譲渡損のとき法人 TransferTax = 0 を期待、got %.0f", corp.TransferTax)
	}
	if indiv.TotalTaxBurden < 0 {
		t.Errorf("個人 TotalTaxBurden が負になってはいけない、got %.0f", indiv.TotalTaxBurden)
	}
	if corp.TotalTaxBurden < 0 {
		t.Errorf("法人 TotalTaxBurden が負になってはいけない、got %.0f", corp.TotalTaxBurden)
	}
}
