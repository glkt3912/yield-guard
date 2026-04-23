package domain

import (
	"math"
	"testing"
)

var baseInput = InvestmentInput{
	LandPrice:      10_000_000,
	BuildingCost:   20_000_000,
	MonthlyRent:    150_000,
	VacancyRate:    0.05,
	LoanAmount:     25_000_000,
	AnnualLoanRate: 0.015,
	LoanYears:      35,
	BuildingType:   BuildingTypeRC,
	ExpenseRate:    0.20,
	IncomeTaxRate:  0.33,
	HoldingYears:   10,
	ExitYieldTarget: 0.06,
}

func TestMonteCarloSimulate_BasicSmoke(t *testing.T) {
	input := MonteCarloInput{
		Base:        baseInput,
		Simulations: 100,
	}
	result := MonteCarloSimulate(input)

	if result.SimulationCount != 100 {
		t.Errorf("SimulationCount = %d, want 100", result.SimulationCount)
	}
	if result.DeadCrossRate < 0 || result.DeadCrossRate > 1 {
		t.Errorf("DeadCrossRate = %.4f, want [0,1]", result.DeadCrossRate)
	}
	if result.SuccessRate < 0 || result.SuccessRate > 1 {
		t.Errorf("SuccessRate = %.4f, want [0,1]", result.SuccessRate)
	}
	if len(result.EquityHistogram) == 0 {
		t.Error("EquityHistogram is empty")
	}
	if result.EquityPercentiles.P50 == 0 {
		t.Error("EquityPercentiles.P50 should be non-zero for valid input")
	}
}

func TestMonteCarloSimulate_DefaultsApplied(t *testing.T) {
	// Simulations=0 はデフォルト1000に補正されること
	input := MonteCarloInput{Base: baseInput, Simulations: 0}
	result := MonteCarloSimulate(input)
	if result.SimulationCount != defaultSimulations {
		t.Errorf("SimulationCount = %d, want %d", result.SimulationCount, defaultSimulations)
	}
}

func TestMonteCarloSimulate_MaxSimulationsCapped(t *testing.T) {
	input := MonteCarloInput{Base: baseInput, Simulations: 99999}
	result := MonteCarloSimulate(input)
	if result.SimulationCount != maxSimulations {
		t.Errorf("SimulationCount = %d, want capped at %d", result.SimulationCount, maxSimulations)
	}
}

func TestCalcIRR_AllNegativeCF(t *testing.T) {
	// 全年 CF が負 → IRR は解なし → NaN
	in := InvestmentInput{
		LandPrice:      1_000_000_000, // 土地10億（意図的に過大）
		BuildingCost:   100_000,
		MonthlyRent:    1,             // 賃料1円
		VacancyRate:    0.99,
		LoanAmount:     0,
		AnnualLoanRate: 0,
		LoanYears:      35,
		BuildingType:   BuildingTypeWood,
		ExpenseRate:    0.20,
		IncomeTaxRate:  0.33,
		HoldingYears:   10,
		ExitYieldTarget: 0.06,
	}
	in.Defaults()
	result := Analyze(in)
	irr := calcIRR(result, in)
	if !math.IsNaN(irr) {
		// 極端なケースでは NaN でなく非常に低い IRR を返す場合もあるが
		// 少なくとも 0 以下であること
		if irr >= 0 {
			t.Errorf("IRR = %.4f, expected negative or NaN for all-negative CF scenario", irr)
		}
	}
}

func TestCalcIRR_ReasonableCase(t *testing.T) {
	in := baseInput
	in.Defaults()
	result := Analyze(in)
	irr := calcIRR(result, in)

	// 健全な投資物件なので IRR は NaN でなく有限値であること
	if math.IsNaN(irr) {
		t.Error("calcIRR returned NaN for a reasonable investment input")
	}
	// IRR は -100% 〜 +1000% の範囲に収まること
	if irr < -0.99 || irr > 10.0 {
		t.Errorf("calcIRR = %.4f, out of expected range", irr)
	}
}

func TestBuildHistogram_SingleValue(t *testing.T) {
	data := []float64{5.0, 5.0, 5.0}
	bins := buildHistogram(data, 20)
	if len(bins) != 1 {
		t.Errorf("single-value histogram should have 1 bin, got %d", len(bins))
	}
	if bins[0].Count != 3 {
		t.Errorf("bin count = %d, want 3", bins[0].Count)
	}
}

func TestBuildPercentiles_Basic(t *testing.T) {
	data := make([]float64, 100)
	for i := range data {
		data[i] = float64(i + 1)
	}
	pct := buildPercentiles(data)
	if pct.P50 < 49 || pct.P50 > 51 {
		t.Errorf("P50 = %.1f, want ~50", pct.P50)
	}
	if pct.P10 < 9 || pct.P10 > 11 {
		t.Errorf("P10 = %.1f, want ~10", pct.P10)
	}
}
