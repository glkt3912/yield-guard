package domain

import "math"

// CalcLTVSensitivity は指定の LTV 水準ごとに DSCR・CF・CF利回りを算出して返す。
// ltvRange が nil の場合は [0.5, 0.6, 0.7, 0.8, 0.9] を使用する。
func CalcLTVSensitivity(input InvestmentInput, ltvRange []float64) []LTVSensitivityRow {
	input.Defaults()
	if ltvRange == nil {
		ltvRange = []float64{0.5, 0.6, 0.7, 0.8, 0.9}
	}

	baseMiscExpenses := (input.LandPrice + input.BuildingCost) * input.MiscExpenseRate
	baseCost := input.LandPrice + input.BuildingCost + baseMiscExpenses
	if baseCost <= 0 {
		return nil
	}

	effectiveVacancy := math.Min(input.VacancyRate, 0.99)
	annualRent := input.MonthlyRent * 12 * (1 - effectiveVacancy)
	annualExpenses := annualRent*input.ExpenseRate + input.AnnualPropertyTax
	noi := annualRent - annualExpenses

	rate1 := resolveRateForYear(input.AnnualLoanRate, input.LoanRateDelta, input.RateAdjustmentSchedule, 1)

	rows := make([]LTVSensitivityRow, 0, len(ltvRange))
	for _, ltv := range ltvRange {
		loanAmount := baseCost * ltv
		loanFee := loanAmount * input.LoanFeeRate
		totalInvestment := baseCost + loanFee
		equity := totalInvestment - loanAmount

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
// YieldTarget（8%境界）は物件価格ベースの表面利回りで判定するため、逆算も物件価格基準で行う。
func calcRequiredForTarget(input InvestmentInput, propertyPrice float64) (requiredRent, costReduction float64) {
	target := input.YieldTarget
	requiredAnnualRent := propertyPrice * target
	requiredRent = requiredAnnualRent / 12

	currentAnnualRent := input.MonthlyRent * 12
	requiredPropertyPrice := currentAnnualRent / target

	excess := propertyPrice - requiredPropertyPrice
	if excess > 0 {
		costReduction = excess
	}
	return requiredRent, costReduction
}
