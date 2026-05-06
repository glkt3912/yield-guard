package domain

import "math"

// CalcLTVSensitivity は指定の LTV 水準ごとに DSCR・CF・CF利回りを算出して返す。
// ltvRange が nil の場合は [0.5, 0.6, 0.7, 0.8, 0.9] を使用する。
func CalcLTVSensitivity(input InvestmentInput, ltvRange []float64) []LTVSensitivityRow {
	input.Defaults()
	if ltvRange == nil {
		ltvRange = []float64{0.5, 0.6, 0.7, 0.8, 0.9}
	}

	miscExpenses := (input.LandPrice + input.BuildingCost) * input.MiscExpenseRate
	totalInvestment := input.LandPrice + input.BuildingCost + miscExpenses
	if totalInvestment <= 0 {
		return nil
	}

	effectiveVacancy := math.Min(input.VacancyRate, 0.99)
	annualRent := input.MonthlyRent * 12 * (1 - effectiveVacancy)
	annualExpenses := annualRent*input.ExpenseRate + input.AnnualPropertyTax
	noi := annualRent - annualExpenses

	rate1 := resolveRateForYear(input.AnnualLoanRate, input.LoanRateDelta, input.RateAdjustmentSchedule, 1)

	rows := make([]LTVSensitivityRow, 0, len(ltvRange))
	for _, ltv := range ltvRange {
		loanAmount := totalInvestment * ltv
		equity := totalInvestment * (1 - ltv)

		var annualDebtService float64
		if input.LoanMethod == LoanMethodEqualPrincipal && input.LoanYears > 0 {
			totalMonths := input.LoanYears * 12
			monthlyPrincipal := loanAmount / float64(totalMonths)
			yearInterest, yearPrincipal := calcYearlyLoanComponentsEqualPrincipal(
				loanAmount, rate1, monthlyPrincipal,
			)
			annualDebtService = yearInterest + yearPrincipal
		} else {
			mp := calcMonthlyPayment(loanAmount, rate1, input.LoanYears)
			annualDebtService = mp * 12
		}

		dscr := CalcDSCR(noi, annualDebtService)
		annualCF := noi - annualDebtService
		cfYield := annualCF / totalInvestment

		rows = append(rows, LTVSensitivityRow{
			LTV:        ltv,
			Equity:     equity,
			LoanAmount: loanAmount,
			DSCR:       dscr,
			AnnualCF:   annualCF,
			CFYield:    cfYield,
		})
	}
	return rows
}

// calcRequiredForTarget は目標利回り達成に必要な値を逆算する
// costReduction は「土地価格または建築費のいずれか一方」を削減すべき金額を表す
func calcRequiredForTarget(input InvestmentInput, totalInvestment float64) (requiredRent, costReduction float64) {
	target := input.YieldTarget
	requiredAnnualRent := totalInvestment * target
	requiredRent = requiredAnnualRent / 12

	currentAnnualRent := input.MonthlyRent * 12
	requiredTotalInvestment := currentAnnualRent / target

	excess := totalInvestment - requiredTotalInvestment
	if excess > 0 {
		costReduction = excess
	}
	return requiredRent, costReduction
}
