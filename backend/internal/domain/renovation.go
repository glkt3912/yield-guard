package domain

const capitalExpenditureThreshold = 600_000.0

// CalcRenovationROI はリフォームROIシミュレーションを実行する
func CalcRenovationROI(input RenovationInput) RenovationResult {
	var totalCost, totalMonthlyRentIncrease float64
	var repairExpenses, capitalExpenditures, virtualLaborCost float64
	classified := make([]ClassifiedRenovationItem, 0, len(input.Items))

	for _, item := range input.Items {
		totalCost += item.Cost
		totalMonthlyRentIncrease += item.ExpectedMonthlyRentIncrease

		isCapEx := item.Cost > capitalExpenditureThreshold
		if isCapEx {
			capitalExpenditures += item.Cost
		} else {
			repairExpenses += item.Cost
		}

		laborCost := 0.0
		if item.IsSelfWork {
			laborCost = item.SelfLaborHours * input.SelfLaborRatePerHour
		}
		virtualLaborCost += laborCost

		classified = append(classified, ClassifiedRenovationItem{
			RenovationItem:       item,
			IsCapitalExpenditure: isCapEx,
			VirtualLaborCost:     laborCost,
		})
	}

	annualRentIncrease := totalMonthlyRentIncrease * 12
	recoveryYears := 0.0
	if annualRentIncrease > 0 {
		recoveryYears = totalCost / annualRentIncrease
	}

	taxSavings := repairExpenses * input.EffectiveTaxRate

	actualYield := 0.0
	denominator := input.PropertyPrice + totalCost
	if denominator > 0 {
		actualYield = (input.AnnualBaseRent + annualRentIncrease - input.AnnualExpenses) / denominator
	}

	return RenovationResult{
		RecoveryYears:       recoveryYears,
		TaxSavings:          taxSavings,
		VirtualLaborCost:    virtualLaborCost,
		CapitalExpenditures: capitalExpenditures,
		RepairExpenses:      repairExpenses,
		ActualYield:         actualYield,
		TotalRenovationCost: totalCost,
		AnnualRentIncrease:  annualRentIncrease,
		ClassifiedItems:     classified,
	}
}
