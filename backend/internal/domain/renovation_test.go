package domain

import "testing"

func TestCalcRenovationROI_BasicCase(t *testing.T) {
	input := RenovationInput{
		PropertyPrice:    10_000_000,
		AnnualBaseRent:   1_200_000,
		AnnualExpenses:   240_000,
		EffectiveTaxRate: 0.30,
		Items: []RenovationItem{
			{Name: "内装", Cost: 300_000, ExpectedMonthlyRentIncrease: 5_000},
		},
	}
	result := CalcRenovationROI(input)

	if !approxEqual(result.AnnualRentIncrease, 60_000, 1) {
		t.Errorf("AnnualRentIncrease = %.0f, want 60000", result.AnnualRentIncrease)
	}
	if !approxEqual(result.RecoveryYears, 5.0, 0.001) {
		t.Errorf("RecoveryYears = %.2f, want 5.0", result.RecoveryYears)
	}
	if !approxEqual(result.TaxSavings, 90_000, 1) {
		t.Errorf("TaxSavings = %.0f, want 90000", result.TaxSavings)
	}
	wantYield := (1_200_000 + 60_000 - 240_000) / float64(10_000_000+300_000)
	if !approxEqual(result.ActualYield, wantYield, 0.0001) {
		t.Errorf("ActualYield = %.4f, want %.4f", result.ActualYield, wantYield)
	}
}

func TestCalcRenovationROI_CapitalExpenditure(t *testing.T) {
	input := RenovationInput{
		PropertyPrice:    10_000_000,
		AnnualBaseRent:   1_200_000,
		AnnualExpenses:   0,
		EffectiveTaxRate: 0.30,
		Items: []RenovationItem{
			{Name: "外壁塗装", Cost: 800_000, ExpectedMonthlyRentIncrease: 10_000},
		},
	}
	result := CalcRenovationROI(input)

	if !approxEqual(result.CapitalExpenditures, 800_000, 1) {
		t.Errorf("CapitalExpenditures = %.0f, want 800000", result.CapitalExpenditures)
	}
	if result.RepairExpenses != 0 {
		t.Errorf("RepairExpenses = %.0f, want 0", result.RepairExpenses)
	}
	if result.TaxSavings != 0 {
		t.Errorf("TaxSavings = %.0f, want 0 (capital expenditure has no immediate deduction)", result.TaxSavings)
	}
	if result.ClassifiedItems[0].IsCapitalExpenditure != true {
		t.Error("expected IsCapitalExpenditure=true for 80万円 item")
	}
}

func TestCalcRenovationROI_MixedItems(t *testing.T) {
	input := RenovationInput{
		PropertyPrice:    10_000_000,
		AnnualBaseRent:   1_200_000,
		AnnualExpenses:   0,
		EffectiveTaxRate: 0.20,
		Items: []RenovationItem{
			{Name: "内装", Cost: 300_000, ExpectedMonthlyRentIncrease: 3_000},
			{Name: "外壁", Cost: 700_000, ExpectedMonthlyRentIncrease: 8_000},
		},
	}
	result := CalcRenovationROI(input)

	if !approxEqual(result.TotalRenovationCost, 1_000_000, 1) {
		t.Errorf("TotalRenovationCost = %.0f, want 1000000", result.TotalRenovationCost)
	}
	if !approxEqual(result.RepairExpenses, 300_000, 1) {
		t.Errorf("RepairExpenses = %.0f, want 300000", result.RepairExpenses)
	}
	if !approxEqual(result.CapitalExpenditures, 700_000, 1) {
		t.Errorf("CapitalExpenditures = %.0f, want 700000", result.CapitalExpenditures)
	}
	if !approxEqual(result.TaxSavings, 60_000, 1) { // 300000 * 0.20
		t.Errorf("TaxSavings = %.0f, want 60000", result.TaxSavings)
	}
	if len(result.ClassifiedItems) != 2 {
		t.Errorf("ClassifiedItems len = %d, want 2", len(result.ClassifiedItems))
	}
}

func TestCalcRenovationROI_SelfWork(t *testing.T) {
	input := RenovationInput{
		SelfLaborRatePerHour: 3_000,
		Items: []RenovationItem{
			{Name: "DIY塗装", Cost: 50_000, ExpectedMonthlyRentIncrease: 0, IsSelfWork: true, SelfLaborHours: 10},
		},
	}
	result := CalcRenovationROI(input)

	if !approxEqual(result.VirtualLaborCost, 30_000, 1) {
		t.Errorf("VirtualLaborCost = %.0f, want 30000", result.VirtualLaborCost)
	}
	if !approxEqual(result.ClassifiedItems[0].VirtualLaborCost, 30_000, 1) {
		t.Errorf("ClassifiedItems[0].VirtualLaborCost = %.0f, want 30000", result.ClassifiedItems[0].VirtualLaborCost)
	}
}

func TestCalcRenovationROI_ZeroRentIncrease(t *testing.T) {
	input := RenovationInput{
		PropertyPrice: 10_000_000,
		Items: []RenovationItem{
			{Name: "補修", Cost: 200_000, ExpectedMonthlyRentIncrease: 0},
		},
	}
	result := CalcRenovationROI(input)
	if result.RecoveryYears != 0 {
		t.Errorf("RecoveryYears = %.2f, want 0 (no rent increase → no recovery)", result.RecoveryYears)
	}
}

func TestCalcRenovationROI_ZeroDenominator(t *testing.T) {
	input := RenovationInput{
		PropertyPrice:  0,
		AnnualBaseRent: 0,
		AnnualExpenses: 0,
		Items:          []RenovationItem{},
	}
	result := CalcRenovationROI(input)
	if result.ActualYield != 0 {
		t.Errorf("ActualYield = %.4f, want 0 for zero denominator", result.ActualYield)
	}
}

func TestCalcRenovationROI_ThresholdBoundary(t *testing.T) {
	at := RenovationInput{
		Items: []RenovationItem{{Name: "工事A", Cost: 600_000}},
	}
	above := RenovationInput{
		Items: []RenovationItem{{Name: "工事B", Cost: 600_001}},
	}
	rAt := CalcRenovationROI(at)
	rAbove := CalcRenovationROI(above)

	if rAt.ClassifiedItems[0].IsCapitalExpenditure {
		t.Error("cost=600000 should be RepairExpense (not capital expenditure)")
	}
	if !rAbove.ClassifiedItems[0].IsCapitalExpenditure {
		t.Error("cost=600001 should be CapitalExpenditure")
	}
}

func TestCalcRenovationROI_ClassifiedItemsLength(t *testing.T) {
	input := RenovationInput{
		Items: []RenovationItem{
			{Name: "A", Cost: 100_000},
			{Name: "B", Cost: 200_000},
			{Name: "C", Cost: 900_000},
		},
	}
	result := CalcRenovationROI(input)
	if len(result.ClassifiedItems) != 3 {
		t.Errorf("ClassifiedItems len = %d, want 3", len(result.ClassifiedItems))
	}
}
