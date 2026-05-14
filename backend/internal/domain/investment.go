package domain

import (
	"context"
	"fmt"
	"sort"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	domainTracerName = "domain"

	// SqmPerTsubo は 1坪あたりの平方メートル数（mlit パッケージからも参照）
	SqmPerTsubo = 3.30578

	LoanMethodEqualPayment   = "equal-payment"  // 元利均等返済
	LoanMethodEqualPrincipal = "equal-principal" // 元金均等返済

	DepreciationMethodStraightLine     = "straight-line"     // 定額法
	DepreciationMethodDecliningBalance = "declining-balance" // 定率法
)

// Analyze は投資入力値から収支シミュレーション結果を算出する
func Analyze(ctx context.Context, input InvestmentInput) InvestmentResult {
	ctx, span := otel.Tracer(domainTracerName).Start(ctx, "domain.Analyze")
	defer span.End()
	span.SetAttributes(
		attribute.Float64("domain.land_price", input.LandPrice),
		attribute.Float64("domain.building_cost", input.BuildingCost),
		attribute.Float64("domain.loan_amount", input.LoanAmount),
	)

	input.Defaults()

	yp := initYieldParams(input)
	requiredRent, landDrop := calcRequiredForTarget(input, yp.totalInvestment)
	lp := initLoanParams(input)
	dp := initDepreciationParams(input)

	// シミュレーション期間: max(LoanYears, HoldingYears, 35)
	//   - LoanYears: ローン完済まで元金返済額が正確に追える
	//   - HoldingYears: 売却試算年が範囲内に収まる
	//   - 35: フロントのグラフ表示が35年固定のため最低35年分を保証
	years := input.LoanYears
	if input.HoldingYears > years {
		years = input.HoldingYears
	}
	if years < 35 {
		years = 35
	}

	sim := simulateYears(input, years, yp, lp, dp)

	dscr := 0.0
	if len(sim.yearlyResults) > 0 {
		noi := sim.yearlyResults[0].AnnualRent - sim.yearlyResults[0].AnnualExpenses
		dscr = CalcDSCR(noi, sim.yearlyResults[0].AnnualLoanPayment)
	}

	exitSalePrice, exitCapGain, exitTax, exitNet, exitEquity := calcExit(
		input, sim.yearlyResults, sim.accumulatedDepreciation, yp.miscExpenses,
	)

	criticalErrors := calcCriticalErrors(input, sim.deadCrossYear, dp.usefulLife)
	stressScenarios := calcAllStressScenarios(ctx, input)
	acquisitionCosts := CalcAcquisitionCosts(
		input.LandPrice,
		input.BuildingCost,
		AcquisitionCostOptions{
			BrokerageMultiplier: 1.0,
			LoanAmount:          input.LoanAmount,
		},
	)
	yieldScenarios := calcYieldScenarios(input, yp.totalInvestment)
	ltvSensitivity := CalcLTVSensitivity(input, nil)

	equity := yp.totalInvestment - input.LoanAmount
	irrNPV := calcIRRNPV(input, sim.yearlyResults, equity, exitNet,
		exitSalePrice, sim.accumulatedDepreciation, yp.miscExpenses)

	multiExit := calcMultiExitComparison(input, sim.yearlyResults, yp.miscExpenses)

	return InvestmentResult{
		TotalInvestment:       yp.totalInvestment,
		MiscExpenses:          yp.miscExpenses,
		GrossYield:            yp.grossYield,
		NetYield:              yp.netYield,
		IsAboveYieldTarget:    yp.grossYield >= input.YieldTarget,
		YieldTarget:           input.YieldTarget,
		RequiredCostReduction: landDrop,
		RequiredMonthlyRent:   requiredRent,
		DeadCrossYear:         sim.deadCrossYear,
		YearlyResults:         sim.yearlyResults,
		CriticalErrors:        criticalErrors,
		AcquisitionCosts:      acquisitionCosts,
		ExitSalePrice:         exitSalePrice,
		ExitCapitalGain:       exitCapGain,
		ExitTransferTax:       exitTax,
		ExitNetProceeds:       exitNet,
		ExitTotalEquity:       exitEquity,
		StressScenarios:       stressScenarios,
		YieldScenarios:        yieldScenarios,
		DSCR:                  dscr,
		LTVSensitivity:        ltvSensitivity,
		IRR:                   irrNPV.irr,
		NPV:                   irrNPV.npv,
		TotalInterest:         sim.totalInterest,
		MultiExitComparison:   multiExit,
	}
}

// calcCriticalErrors はPhase 1の重大リスク項目を判定する
func calcCriticalErrors(input InvestmentInput, deadCrossYear, usefulLife int) []CriticalError {
	var errs []CriticalError

	// LAND_VALUE_GUARD: 積算評価額が購入総額の50%未満
	// 積算評価 = 土地価格 + 建物費用 × (残存耐用年数 / 法定耐用年数)
	totalPurchase := input.LandPrice + input.BuildingCost
	residualLife := CalcResidualUsefulLife(input.BuildingType, input.BuildingAge)
	buildingAppraisedValue := input.BuildingCost * float64(residualLife) / float64(usefulLife)
	appraisedValue := input.LandPrice + buildingAppraisedValue
	if totalPurchase > 0 && appraisedValue < totalPurchase*0.5 {
		errs = append(errs, CriticalError{
			Code:   "LAND_VALUE_GUARD",
			Status: CriticalStatusReject,
			Message: fmt.Sprintf(
				"積算評価額（%.0f万円）が購入総額（%.0f万円）の50%%未満です。"+
					"銀行の担保評価が低くなり次の買主がローンを組めない可能性があります。",
				appraisedValue/10000, totalPurchase/10000,
			),
		})
	}

	// DEADCROSS_EARLY: デッドクロスが10年以内に発生
	if deadCrossYear > 0 && deadCrossYear <= 10 {
		errs = append(errs, CriticalError{
			Code:   "DEADCROSS_EARLY",
			Status: CriticalStatusReject,
			Message: fmt.Sprintf(
				"%d年目にデッドクロスが発生します。帳簿上は黒字でもキャッシュ不足に陥るリスクがあります。",
				deadCrossYear,
			),
		})
	}

	if errs == nil {
		errs = []CriticalError{}
	}
	return errs
}

// CalcDSCR は借入金償還余裕率を返す（DSCR = NOI / 年間返済額）
// annualDebtService が 0 以下の場合は 0 を返す。
func CalcDSCR(noi, annualDebtService float64) float64 {
	if annualDebtService <= 0 {
		return 0
	}
	return noi / annualDebtService
}

// CalcLandPriceStats は取引データから統計を算出する
func CalcLandPriceStats(ctx context.Context, transactions []LandTransaction) LandPriceStats {
	_, span := otel.Tracer(domainTracerName).Start(ctx, "domain.CalcLandPriceStats")
	defer span.End()
	span.SetAttributes(attribute.Int("domain.transaction_count", len(transactions)))

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

	zoning := calcZoningSummary(transactions)
	return LandPriceStats{
		Count:          len(prices),
		AverageTsubo:   avg,
		MedianTsubo:    median,
		MinTsubo:       prices[0],
		MaxTsubo:       prices[len(prices)-1],
		Transactions:   transactions,
		LowDataWarning: len(prices) < lowDataThreshold,
		WarningMessage: warning,
		Zoning:         zoning,
		UrbanRisks:     detectUrbanRisks(transactions, zoning),
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
