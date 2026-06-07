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
		input, sim.yearlyResults, sim.accumulatedDepreciation,
	)

	criticalErrors := calcCriticalErrors(input, sim.deadCrossYear, dp.usefulLife)
	stressScenarios := calcAllStressScenarios(ctx, input)
	isNewBuilding := input.BuildingAge == 0
	if input.IsFirstRegistration != nil {
		isNewBuilding = *input.IsFirstRegistration
	}
	acquisitionCosts := CalcAcquisitionCosts(
		input.LandPrice,
		input.BuildingCost,
		AcquisitionCostOptions{
			BrokerageMultiplier: 1.0,
			LoanAmount:          input.LoanAmount,
			IsNewBuilding:       isNewBuilding,
		},
	)
	yieldScenarios := calcYieldScenarios(input, yp.totalInvestment)
	ltvSensitivity := CalcLTVSensitivity(input, nil)

	equity := yp.totalInvestment - input.LoanAmount
	irrNPV := calcIRRNPV(input, sim.yearlyResults, equity, exitNet)

	multiExit := calcMultiExitComparison(input, sim.yearlyResults, yp.miscExpenses)
	taxSim := CalcTaxSimulation(input, sim.yearlyResults, exitCapGain)

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
		TaxSimulation:         taxSim,
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

	// OVERLOAN: ローン金額が土地取得費＋建築費を超過
	if input.LoanAmount > input.LandPrice+input.BuildingCost {
		errs = append(errs, CriticalError{
			Code:   "OVERLOAN",
			Status: CriticalStatusWarning,
			Message: fmt.Sprintf(
				"ローン金額（%.0f万円）が物件取得費（%.0f万円）を超えています。"+
					"オーバーローンは審査否決や担保割れリスクがあります。",
				input.LoanAmount/10000, (input.LandPrice+input.BuildingCost)/10000,
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

// CalcYieldDifficulty は坪単価中央値・予算・目標利回りから利回り達成難易度を返す。
// difficulty は "achievable" | "slightly-difficult" | "difficult"。
// medianTsubo が 0 の場合は ("", "") を返す。
func CalcYieldDifficulty(medianTsubo, budget, targetYield float64) (difficulty, label string) {
	if medianTsubo <= 0 {
		return "", ""
	}
	var rentPerTsubo float64
	if budget > 0 {
		// budget / medianTsubo = 購入可能坪数。エリアごとに坪単価が異なるため面積は変わる。
		// rentPerTsubo = budget × yield / 12 / (budget/medianTsubo) = medianTsubo × yield / 12
		tsuboCount := budget / medianTsubo
		if tsuboCount < 1 {
			tsuboCount = 1
		}
		rentPerTsubo = budget * targetYield / 12 / tsuboCount
	} else {
		totalCostEst := medianTsubo*30 + 10_000_000
		rentPerTsubo = totalCostEst * targetYield / 12 / 30
	}
	switch {
	case rentPerTsubo <= 8000:
		return "achievable", "達成可能"
	case rentPerTsubo <= 15000:
		return "slightly-difficult", "やや困難"
	default:
		return "difficult", "困難"
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

// landTrendMinSamples は年次比較に必要な最小取引件数。
// どちらかの年がこの件数を下回る場合は信頼性が低いため "不明" を返す。
const landTrendMinSamples = 2

// CalcLandPriceTrend は取引データを年次グルーピングして坪単価変化率を算出し
// "上昇" | "安定" | "下落" | "不明" を返す。
// 直近2年分の有効データがない、またはどちらかの年の件数が landTrendMinSamples 未満の場合は "不明" を返す。
func CalcLandPriceTrend(transactions []LandTransaction) string {
	byYear := map[int][]float64{}
	for _, t := range transactions {
		if t.PricePerTsubo <= 0 {
			continue
		}
		y := parsePeriodYear(t.Period)
		if y == 0 {
			continue
		}
		byYear[y] = append(byYear[y], t.PricePerTsubo)
	}
	if len(byYear) < 2 {
		return "不明"
	}

	years := make([]int, 0, len(byYear))
	for y := range byYear {
		years = append(years, y)
	}
	sort.Ints(years)

	recentYear := years[len(years)-1]
	prevYear := years[len(years)-2]

	if len(byYear[recentYear]) < landTrendMinSamples || len(byYear[prevYear]) < landTrendMinSamples {
		return "不明"
	}

	recentMedian := medianFloat64(byYear[recentYear])
	prevMedian := medianFloat64(byYear[prevYear])
	if prevMedian == 0 {
		return "不明"
	}

	changeRate := (recentMedian - prevMedian) / prevMedian * 100
	switch {
	case changeRate > 5:
		return "上昇"
	case changeRate < -5:
		return "下落"
	default:
		return "安定"
	}
}

// parsePeriodYear は国交省 API の取引時点文字列から西暦年を返す。
// 例: "令和5年第3四半期" → 2023, "2024年第1四半期" → 2024
func parsePeriodYear(s string) int {
	eraMap := []struct {
		prefix string
		base   int
	}{
		{"令和", 2018},
		{"平成", 1988},
		{"昭和", 1925},
		{"大正", 1911},
		{"明治", 1867},
	}
	for _, e := range eraMap {
		if len(s) >= len(e.prefix) && s[:len(e.prefix)] == e.prefix {
			rest := s[len(e.prefix):]
			numEnd := 0
			for numEnd < len(rest) && rest[numEnd] >= '0' && rest[numEnd] <= '9' {
				numEnd++
			}
			if numEnd == 0 {
				return 0
			}
			n := 0
			for _, c := range rest[:numEnd] {
				n = n*10 + int(c-'0')
			}
			return e.base + n
		}
	}
	// 西暦形式 "2024年..." または "2024第..."
	numEnd := 0
	for numEnd < len(s) && s[numEnd] >= '0' && s[numEnd] <= '9' {
		numEnd++
	}
	if numEnd == 4 {
		n := 0
		for _, c := range s[:4] {
			n = n*10 + int(c-'0')
		}
		if n >= 1900 && n <= 2100 {
			return n
		}
	}
	return 0
}

func medianFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	cp := make([]float64, len(vals))
	copy(cp, vals)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 0 {
		return (cp[n/2-1] + cp[n/2]) / 2
	}
	return cp[n/2]
}
