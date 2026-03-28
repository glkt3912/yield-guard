package domain

import (
	"fmt"
	"math"
	"sort"
)

const (
	// SqmPerTsubo は 1坪あたりの平方メートル数（mlit パッケージからも参照）
	SqmPerTsubo     = 3.30578
	targetYield8pct = 0.08 // 8%境界線
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
	requiredRent, landDrop := calcRequired8pct(input, totalInvestment)

	// ローン月次計算
	monthlyPayment := calcMonthlyPayment(input.LoanAmount, effectiveRate, input.LoanYears)

	// 減価償却 (定額法)
	// 中古物件は簡便法耐用年数を使用（新築は法定耐用年数）
	usefulLife := CalcResidualUsefulLife(input.BuildingType, input.BuildingAge)
	annualDepreciation := input.BuildingCost / float64(usefulLife)

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

		// デッドクロス判定: 元金返済額 > 減価償却費 となるゾーン
		// 耐用年数経過後は減価償却=0のため、元金返済が残っていれば即デッドクロス
		// ローン完済後（annualPrincipal==0）はデッドクロスゾーンから脱出
		inDeadCrossZone := annualPrincipal > 0 && annualPrincipal > yearDepreciation
		isDeadCrossYear := false
		if deadCrossYear == -1 && inDeadCrossZone {
			deadCrossYear = year
			isDeadCrossYear = true
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
			IsDeadCrossYear:      isDeadCrossYear,
			IsInDeadCrossZone:    inDeadCrossZone,
		}
	}

	// 出口戦略 (holdingYears 年後に売却)
	exitSalePrice, exitCapGain, exitTax, exitNet, exitEquity := calcExit(
		input, yearlyResults, accumulatedDepreciation, miscExpenses,
	)

	return InvestmentResult{
		TotalInvestment:       totalInvestment,
		MiscExpenses:          miscExpenses,
		GrossYield:            grossYield,
		NetYield:              netYield,
		IsAbove8Percent:       grossYield >= targetYield8pct,
		RequiredCostReduction: landDrop,
		RequiredMonthlyRent:   requiredRent,
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
// 最終月で月次返済額が残高を超える場合は残高のみを元金返済として扱い、誤差を防ぐ
func calcYearlyLoanComponents(balance, annualRate, monthlyPayment float64) (interest, principal float64) {
	if annualRate == 0 {
		if monthlyPayment*12 > balance {
			return 0, balance
		}
		return 0, monthlyPayment * 12
	}
	r := annualRate / 12
	remaining := balance
	for range 12 {
		if remaining <= 0 {
			break
		}
		monthInterest := remaining * r
		monthPrincipal := monthlyPayment - monthInterest
		// 最終月: 残高が月次元金返済より少ない場合は残高のみ返済
		if monthPrincipal > remaining {
			monthPrincipal = remaining
		}
		interest += monthInterest
		principal += monthPrincipal
		remaining -= monthPrincipal
	}
	return interest, principal
}

// calcRequired8pct は表面利回り8%達成に必要な値を逆算する
// costReduction は「土地価格または建築費のいずれか一方」を削減すべき金額を表す
func calcRequired8pct(input InvestmentInput, totalInvestment float64) (requiredRent, costReduction float64) {
	// 目標利回り8%達成に必要な月額賃料
	requiredAnnualRent := totalInvestment * targetYield8pct
	requiredRent = requiredAnnualRent / 12

	// 現在の賃料で8%達成に必要な総投資額
	currentAnnualRent := input.MonthlyRent * 12
	requiredTotalInvestment := currentAnnualRent / targetYield8pct

	// 過剰投資額 = 土地 or 建築費いずれかを削減すべき額
	excess := totalInvestment - requiredTotalInvestment
	if excess > 0 {
		costReduction = excess
	}
	return requiredRent, costReduction
}

// 譲渡所得税率（所得税＋復興特別所得税＋住民税）
// 根拠: 租税特別措置法31条・32条、復興財源確保法33条（2037年まで）
const (
	shortTermTransferTaxRate  = 0.3963  // 短期（5年以下）: 30.63% + 9%
	longTermTransferTaxRate   = 0.20315 // 長期（5年超）: 15.315% + 5%
	longTerm10YrTransferTaxRate = 0.14210 // 長期（10年超、6000万円以下部分）: 10.21% + 4%
)

// calcExit は出口戦略（売却）の試算を行う
//
// 売却価格: NOI（純収益）/ 目標利回り（実質ベース）で収益還元法により算出
// 取得費: 土地 + 建物簿価 + 取得時諸経費（税法上の取得費）
// 売却費用: 仲介手数料の上限額を概算控除（消費税込み）
// 税率: 保有5年超で長期、10年超で軽減税率を適用
func calcExit(input InvestmentInput, yearly []YearlyResult, accumulatedDepreciation float64, miscExpenses float64) (
	salePrice, capitalGain, transferTax, netProceeds, totalEquity float64,
) {
	if len(yearly) == 0 || input.HoldingYears <= 0 || input.ExitYieldTarget <= 0 {
		return
	}

	holdIdx := input.HoldingYears - 1
	if holdIdx >= len(yearly) {
		holdIdx = len(yearly) - 1
	}

	exitYear := yearly[holdIdx]

	// 収益還元法: 売却価格 = NOI / 目標利回り（実質ベース）
	// NOI = 実効賃料収入 - 運営経費（ローン利息は含まない）
	noi := exitYear.AnnualRent - exitYear.AnnualExpenses
	salePrice = noi / input.ExitYieldTarget

	// 売却費用（仲介手数料上限額の概算・消費税込み）
	// 根拠: 宅建業法46条 上限 = 売却価格×3%+6万円（+消費税10%）
	sellExpenses := (salePrice*0.03+60_000) * 1.10

	// 建物の税務上の簿価（定額法累計控除後）
	bookValueBuilding := input.BuildingCost - accumulatedDepreciation
	if bookValueBuilding < 0 {
		bookValueBuilding = 0
	}

	// 取得費 = 土地取得費 + 建物簿価 + 取得時諸経費
	// 根拠: 所得税法38条（取得費に含まれる付随費用）
	acquisitionCost := input.LandPrice + bookValueBuilding + miscExpenses

	// 譲渡所得 = 売却価格 - 売却費用 - 取得費
	capitalGain = salePrice - sellExpenses - acquisitionCost

	if capitalGain > 0 {
		var taxRate float64
		switch {
		case input.HoldingYears > 10:
			taxRate = longTerm10YrTransferTaxRate // 軽減税率（6000万円超部分は通常長期税率だが簡略化）
		case input.HoldingYears > 5:
			taxRate = longTermTransferTaxRate
		default:
			taxRate = shortTermTransferTaxRate
		}
		transferTax = capitalGain * taxRate
	}

	netProceeds = salePrice - sellExpenses - transferTax - exitYear.RemainingLoanBalance
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

	const lowDataThreshold = 10
	warning := ""
	if len(prices) < lowDataThreshold {
		warning = fmt.Sprintf("取引件数が%d件と少ないため統計の信頼性が低い可能性があります", len(prices))
	}

	return LandPriceStats{
		Count:          len(prices),
		AverageTsubo:   avg,
		MedianTsubo:    median,
		MinTsubo:       prices[0],
		MaxTsubo:       prices[len(prices)-1],
		Transactions:   transactions,
		LowDataWarning: len(prices) < lowDataThreshold,
		WarningMessage: warning,
	}
}

// CompareLandPrice は検討中の土地価格と相場を比較する
func CompareLandPrice(stats LandPriceStats, landPrice, areaSqm float64) LandPriceComparison {
	tsubo := areaSqm / SqmPerTsubo
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
	return sqm / SqmPerTsubo
}

// TsuboToSqm は坪を平方メートルに変換する
func TsuboToSqm(tsubo float64) float64 {
	return tsubo * SqmPerTsubo
}
