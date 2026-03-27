package domain

import (
	"math"
	"sort"
)

const (
	sqmPerTsubo    = 3.30578  // 1坪 = 3.30578 m²
	targetYield8pct = 0.08    // 8%境界線
)

// Analyze は投資入力値から収支シミュレーション結果を算出する
func Analyze(input InvestmentInput) InvestmentResult {
	input.Defaults()

	// ストレステスト値を適用
	effectiveVacancy := input.VacancyRate + input.VacancyRateDelta
	effectiveRate := input.AnnualLoanRate + input.LoanRateDelta

	miscExpenses := (input.LandPrice + input.BuildingCost) * input.MiscExpenseRate
	totalInvestment := input.LandPrice + input.BuildingCost + miscExpenses

	annualRent := input.MonthlyRent * 12 * (1 - effectiveVacancy)
	grossYield := 0.0
	if totalInvestment > 0 {
		grossYield = (input.MonthlyRent * 12) / totalInvestment
	}

	annualExpenses := annualRent * input.ExpenseRate
	netYield := 0.0
	if totalInvestment > 0 {
		netYield = (annualRent - annualExpenses) / totalInvestment
	}

	// 8%逆算
	requiredRent, landDrop, buildingDrop := calcRequired8pct(input, totalInvestment)

	// ローン月次計算
	monthlyPayment := calcMonthlyPayment(input.LoanAmount, effectiveRate, input.LoanYears)

	// 減価償却 (定額法)
	usefulLife := input.BuildingType.UsefulLife()
	annualDepreciation := 0.0
	if usefulLife > 0 {
		annualDepreciation = input.BuildingCost / float64(usefulLife)
	}

	// 年次シミュレーション
	years := input.LoanYears
	if input.HoldingYears > years {
		years = input.HoldingYears
	}
	if years < 35 {
		years = 35
	}

	yearlyResults := make([]YearlyResult, years)
	remainingBalance := input.LoanAmount
	cumulativeCF := 0.0
	deadCrossYear := -1
	var accumulatedDepreciation float64

	for y := 0; y < years; y++ {
		year := y + 1
		annualInterest := 0.0
		annualPrincipal := 0.0
		annualLoanPayment := 0.0

		if remainingBalance > 0 && year <= input.LoanYears {
			annualInterest, annualPrincipal = calcYearlyLoanComponents(
				remainingBalance, effectiveRate, monthlyPayment,
			)
			annualLoanPayment = monthlyPayment * 12
			remainingBalance -= annualPrincipal
			if remainingBalance < 0 {
				remainingBalance = 0
			}
		}

		yearAnnualRent := annualRent
		yearExpenses := yearAnnualRent * input.ExpenseRate

		// 減価償却は耐用年数内のみ
		yearDepreciation := 0.0
		if year <= usefulLife {
			yearDepreciation = annualDepreciation
		}
		accumulatedDepreciation += yearDepreciation

		// 課税所得 = 収入 - 利息 - 減価償却 - 経費
		taxableIncome := yearAnnualRent - annualInterest - yearDepreciation - yearExpenses
		incomeTax := 0.0
		if taxableIncome > 0 {
			incomeTax = taxableIncome * input.IncomeTaxRate
		}

		cashFlow := yearAnnualRent - annualLoanPayment - yearExpenses
		afterTaxCF := cashFlow - incomeTax
		cumulativeCF += afterTaxCF

		// デッドクロス判定: 元金返済額 > 減価償却費 となる最初の年
		// 耐用年数経過後は減価償却=0のため、元金返済が残っていれば即デッドクロス
		isDead := false
		if deadCrossYear == -1 && annualPrincipal > 0 && annualPrincipal > yearDepreciation {
			deadCrossYear = year
			isDead = true
		}

		yearlyResults[y] = YearlyResult{
			Year:                 year,
			AnnualRent:           yearAnnualRent,
			AnnualLoanPayment:    annualLoanPayment,
			AnnualInterest:       annualInterest,
			AnnualPrincipal:      annualPrincipal,
			AnnualDepreciation:   yearDepreciation,
			AnnualExpenses:       yearExpenses,
			TaxableIncome:        taxableIncome,
			IncomeTax:            incomeTax,
			CashFlow:             cashFlow,
			AfterTaxCashFlow:     afterTaxCF,
			RemainingLoanBalance: remainingBalance,
			CumulativeCashFlow:   cumulativeCF,
			IsDeadCrossYear:      isDead,
		}
	}

	// 出口戦略 (holdingYears 年後に売却)
	exitSalePrice, exitCapGain, exitTax, exitNet, exitEquity := calcExit(
		input, yearlyResults, accumulatedDepreciation,
	)

	return InvestmentResult{
		TotalInvestment:          totalInvestment,
		MiscExpenses:             miscExpenses,
		GrossYield:               grossYield,
		NetYield:                 netYield,
		IsAbove8Percent:          grossYield >= targetYield8pct,
		RequiredLandPriceDrop:    landDrop,
		RequiredBuildingCostDrop: buildingDrop,
		RequiredMonthlyRent:      requiredRent,
		DeadCrossYear:            deadCrossYear,
		YearlyResults:            yearlyResults,
		ExitSalePrice:            exitSalePrice,
		ExitCapitalGain:          exitCapGain,
		ExitTransferTax:          exitTax,
		ExitNetProceeds:          exitNet,
		ExitTotalEquity:          exitEquity,
	}
}

// calcMonthlyPayment は元利均等返済の月次返済額を計算する
// P: 元金, annualRate: 年利, years: 返済年数
func calcMonthlyPayment(principal, annualRate float64, years int) float64 {
	if principal <= 0 || years <= 0 {
		return 0
	}
	if annualRate == 0 {
		return principal / float64(years*12)
	}
	r := annualRate / 12
	n := float64(years * 12)
	return principal * r * math.Pow(1+r, n) / (math.Pow(1+r, n) - 1)
}

// calcYearlyLoanComponents は1年分の利息・元金返済額を計算する
func calcYearlyLoanComponents(balance, annualRate, monthlyPayment float64) (interest, principal float64) {
	if annualRate == 0 {
		principal = monthlyPayment * 12
		return 0, principal
	}
	r := annualRate / 12
	remaining := balance
	for m := 0; m < 12; m++ {
		monthInterest := remaining * r
		monthPrincipal := monthlyPayment - monthInterest
		interest += monthInterest
		principal += monthPrincipal
		remaining -= monthPrincipal
		if remaining < 0 {
			remaining = 0
		}
	}
	return interest, principal
}

// calcRequired8pct は表面利回り8%達成に必要な値を逆算する
func calcRequired8pct(input InvestmentInput, totalInvestment float64) (requiredRent, landDrop, buildingDrop float64) {
	// 目標利回り8%達成に必要な月額賃料
	requiredAnnualRent := totalInvestment * targetYield8pct
	requiredRent = requiredAnnualRent / 12

	// 現在の賃料で8%達成に必要な総投資額
	currentAnnualRent := input.MonthlyRent * 12
	requiredTotalInvestment := currentAnnualRent / targetYield8pct

	// 土地値または建築費をどれだけ下げるべきか
	excess := totalInvestment - requiredTotalInvestment
	if excess > 0 {
		landDrop = excess
		buildingDrop = excess
	}
	return requiredRent, landDrop, buildingDrop
}

// calcExit は出口戦略（売却）の試算を行う
func calcExit(input InvestmentInput, yearly []YearlyResult, accumulatedDepreciation float64) (
	salePrice, capitalGain, transferTax, netProceeds, totalEquity float64,
) {
	if len(yearly) == 0 || input.HoldingYears <= 0 {
		return
	}

	holdIdx := input.HoldingYears - 1
	if holdIdx >= len(yearly) {
		holdIdx = len(yearly) - 1
	}

	exitYear := yearly[holdIdx]
	annualRent := exitYear.AnnualRent

	salePrice = annualRent / input.ExitYieldTarget

	// 建物の簿価 = 建築費 - 累積減価償却 (土地は含まない)
	bookValueBuilding := input.BuildingCost - accumulatedDepreciation
	if bookValueBuilding < 0 {
		bookValueBuilding = 0
	}
	// 譲渡所得 = 売却価格 - (土地取得費 + 建物簿価)
	capitalGain = salePrice - (input.LandPrice + bookValueBuilding)

	if capitalGain > 0 {
		taxRate := 0.39363 // 短期 (5年以下)
		if input.HoldingYears > 5 {
			taxRate = 0.20315 // 長期
		}
		transferTax = capitalGain * taxRate
	}

	netProceeds = salePrice - transferTax - exitYear.RemainingLoanBalance
	totalEquity = netProceeds + exitYear.CumulativeCashFlow
	return
}

// CalcLandPriceStats は取引データから統計を算出する
func CalcLandPriceStats(transactions []LandTransaction) LandPriceStats {
	if len(transactions) == 0 {
		return LandPriceStats{}
	}

	prices := make([]float64, 0, len(transactions))
	for _, t := range transactions {
		if t.PricePerTsubo > 0 {
			prices = append(prices, t.PricePerTsubo)
		}
	}
	if len(prices) == 0 {
		return LandPriceStats{Transactions: transactions}
	}

	sort.Float64s(prices)

	sum := 0.0
	for _, p := range prices {
		sum += p
	}
	avg := sum / float64(len(prices))

	median := 0.0
	n := len(prices)
	if n%2 == 0 {
		median = (prices[n/2-1] + prices[n/2]) / 2
	} else {
		median = prices[n/2]
	}

	return LandPriceStats{
		Count:        len(prices),
		AverageTsubo: avg,
		MedianTsubo:  median,
		MinTsubo:     prices[0],
		MaxTsubo:     prices[len(prices)-1],
		Transactions: transactions,
	}
}

// CompareLandPrice は検討中の土地価格と相場を比較する
func CompareLandPrice(stats LandPriceStats, landPrice, areaSqm float64) LandPriceComparison {
	tsubo := areaSqm / sqmPerTsubo
	pricePerTsubo := 0.0
	if tsubo > 0 {
		pricePerTsubo = landPrice / tsubo
	}

	diffFromAvg := pricePerTsubo - stats.AverageTsubo
	diffFromMedian := pricePerTsubo - stats.MedianTsubo

	assessment := "相場"
	if diffFromMedian > stats.MedianTsubo*0.10 {
		assessment = "割高"
	} else if diffFromMedian < -stats.MedianTsubo*0.10 {
		assessment = "割安"
	}

	return LandPriceComparison{
		Stats:              stats,
		InputLandPrice:     landPrice,
		InputArea:          areaSqm,
		InputPricePerTsubo: pricePerTsubo,
		DiffFromAverage:    diffFromAvg,
		DiffFromMedian:     diffFromMedian,
		Assessment:         assessment,
	}
}

// SqmToTsubo は平方メートルを坪に変換する
func SqmToTsubo(sqm float64) float64 {
	return sqm / sqmPerTsubo
}

// TsuboToSqm は坪を平方メートルに変換する
func TsuboToSqm(tsubo float64) float64 {
	return tsubo * sqmPerTsubo
}
