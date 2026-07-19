package domain

import (
	"context"
	"math"
	"sync"
)

const dscrSafeThreshold = 1.2

// yieldParams は yield / vacancy の初期計算結果をまとめた内部 struct。
type yieldParams struct {
	miscExpenses     float64
	propertyPrice    float64 // 物件価格（土地+建物、諸費用を含まない）
	totalInvestment  float64
	annualRent       float64
	grossYield       float64 // 総投資利回り（満室想定年収/総投資額。諸費用込み）
	marketGrossYield float64 // 表面利回り（満室想定年収/物件価格[土地+建物]。市場慣行ベース）
	netYield         float64
}

// loanParams はローン初期値をまとめた内部 struct。
// currentRate / monthlyPayment は変動金利スケジュールによりループ内で変化するため、
// ループ開始時にローカル変数へコピーして使う。
type loanParams struct {
	currentRate           float64
	monthlyPayment        float64
	monthlyPrincipalFixed float64 // 元金均等返済のみ非ゼロ
}

// depreciationParams は減価償却初期値をまとめた内部 struct。
// bookValue は定率法でループ内に変化するため、ループ開始時にローカル変数へコピーして使う。
type depreciationParams struct {
	usefulLife         int
	annualDepreciation float64 // 定額法定数。定率法では 0
	bookValue          float64 // 定率法のみ非ゼロ
	decliningRate      float64 // 定率法のみ非ゼロ
}

// simulationResult は年次シミュレーションループの出力をまとめた内部 struct。
type simulationResult struct {
	yearlyResults           []YearlyResult
	accumulatedDepreciation float64
	deadCrossYear           int     // -1 = デッドクロスなし
	totalInterest           float64 // 保有期間の総支払利息
}

func initYieldParams(input InvestmentInput) yieldParams {
	effectiveVacancy := math.Min(input.VacancyRate+input.VacancyRateDelta, 0.99)
	loanFee := input.LoanAmount * input.LoanFeeRate
	propertyPrice := input.LandPrice + input.BuildingCost
	miscExpenses := propertyPrice*input.MiscExpenseRate + loanFee
	totalInvestment := propertyPrice + miscExpenses
	annualRent := input.MonthlyRent * 12 * (1 - effectiveVacancy)
	grossYield := 0.0
	if totalInvestment > 0 {
		grossYield = (input.MonthlyRent * 12) / totalInvestment
	}
	// 表面利回り（市場慣行）は物件価格（土地+建物、諸費用を含まない）が分母。
	// 物件広告・REINS の「表面利回り」と直接比較できる値であり、8%境界線判定にも用いる。
	marketGrossYield := 0.0
	if propertyPrice > 0 {
		marketGrossYield = (input.MonthlyRent * 12) / propertyPrice
	}
	// initYieldParams は初年度の利回り指標を計算する。
	// 経費インフレ率は年次シミュレーション（simulateYears）で複利適用するため、
	// ここでは適用しない（初年度スナップショットとして扱う）。
	annualExpenses := annualRent * calcEffectiveExpenseRate(input)
	netYield := 0.0
	if totalInvestment > 0 {
		netYield = (annualRent - annualExpenses) / totalInvestment
	}
	return yieldParams{
		miscExpenses:     miscExpenses,
		propertyPrice:    propertyPrice,
		totalInvestment:  totalInvestment,
		annualRent:       annualRent,
		grossYield:       grossYield,
		marketGrossYield: marketGrossYield,
		netYield:         netYield,
	}
}

func initLoanParams(input InvestmentInput) loanParams {
	currentRate := resolveRateForYear(input.AnnualLoanRate, input.LoanRateDelta, input.RateAdjustmentSchedule, 1)
	var monthlyPayment, monthlyPrincipalFixed float64
	if input.LoanMethod == LoanMethodEqualPrincipal && input.LoanYears > 0 {
		monthlyPrincipalFixed = input.LoanAmount / float64(input.LoanYears*12)
	} else {
		monthlyPayment = calcMonthlyPayment(input.LoanAmount, currentRate, input.LoanYears)
	}
	return loanParams{
		currentRate:           currentRate,
		monthlyPayment:        monthlyPayment,
		monthlyPrincipalFixed: monthlyPrincipalFixed,
	}
}

func initDepreciationParams(input InvestmentInput) depreciationParams {
	// 中古物件は簡便法耐用年数を使用（新築は法定耐用年数）
	usefulLife := CalcResidualUsefulLife(input.BuildingType, input.BuildingAge)
	annualDepreciation := input.BuildingCost / float64(usefulLife)
	// bookValue / decliningRate は定率法のみで使用する（ループをまたいで簿価を追跡）
	var bookValue, decliningRate float64
	if input.DepreciationMethod == DepreciationMethodDecliningBalance {
		bookValue = input.BuildingCost
		decliningRate = 2.0 / float64(usefulLife)
	}
	return depreciationParams{
		usefulLife:         usefulLife,
		annualDepreciation: annualDepreciation,
		bookValue:          bookValue,
		decliningRate:      decliningRate,
	}
}

// calcEffectiveExpenseRate は詳細経費フィールドの合計が 0 より大きければその合計を返し、
// 合計が 0 の場合は従来の ExpenseRate にフォールバックする（後方互換）。
func calcEffectiveExpenseRate(input InvestmentInput) float64 {
	detail := input.ManagementFeeRate + input.RepairReserveRate + input.InsuranceFeeRate + input.OtherExpenseRate
	if detail > 0 {
		return detail
	}
	return input.ExpenseRate
}

// calcAnnualTurnoverCost は平均入居期間・原状回復費・AD・フリーレントから
// 年間のターンオーバーコストを算出して返す。
// AvgTenancyYears が 0 以下の場合は 0 を返す（後方互換）。
func calcAnnualTurnoverCost(input InvestmentInput, monthlyRent float64) float64 {
	if input.AvgTenancyYears <= 0 {
		return 0
	}
	turnoverPerYear := 1.0 / input.AvgTenancyYears
	freeLeaseLoss := monthlyRent * input.RentFreePeriod * turnoverPerYear
	return (input.RestorationCost+input.AdFee)*turnoverPerYear + freeLeaseLoss
}

// capexForYear は CapexSchedule から指定年の修繕費合計を返す。
func capexForYear(schedule []CapexEvent, year int) float64 {
	total := 0.0
	for _, ev := range schedule {
		if ev.Year == year {
			total += ev.Amount
		}
	}
	return total
}

// rentForYear は賃料上昇・下落シナリオを考慮した N 年目の年間実効賃料を返す。
// y は 0-indexed（1年目 = y=0）。
//
// 設計: y=0 は「初年度（変化なし）」を表す（RentDeclineRate の既存挙動と対称）。
// 上昇期 y=0..rentGrowthYears-1 は (1+g)^y を適用し、y=rentGrowthYears-1 が上昇期最終年。
// y=rentGrowthYears は「上昇期終了直後の最初の下落年」であり、ピーク × (1-d)^1 を返す。
// これは "下落1年目が即始まる保守的モデル" を採用したもの。
// peak は上昇が続いたと仮定した場合の理論値 (1+g)^rentGrowthYears を基準とする。
func rentForYear(baseRent float64, rentDeclineRate, rentGrowthRate float64, rentGrowthYears, y int) float64 {
	if rentGrowthRate <= 0 || rentGrowthYears <= 0 {
		return baseRent * math.Pow(1-rentDeclineRate, float64(y))
	}
	if y < rentGrowthYears {
		return baseRent * math.Pow(1+rentGrowthRate, float64(y))
	}
	peak := baseRent * math.Pow(1+rentGrowthRate, float64(rentGrowthYears))
	return peak * math.Pow(1-rentDeclineRate, float64(y-rentGrowthYears+1))
}

// simulateYears は年次 P&L ループを実行し yearlyResults・デッドクロス年・累積減価償却額を返す。
// years は呼び出し元 Analyze() が決定した値をそのまま受け取る。
func simulateYears(input InvestmentInput, years int, yp yieldParams, lp loanParams, dp depreciationParams) simulationResult {
	yearlyResults := make([]YearlyResult, years)
	remainingBalance := input.LoanAmount
	cumulativeCF := 0.0
	deadCrossYear := -1
	var accumulatedDepreciation float64
	var totalInterest float64

	// loanParams / depreciationParams の可変フィールドをループローカルにコピー
	currentRate := lp.currentRate
	monthlyPayment := lp.monthlyPayment
	bookValue := dp.bookValue

	for y := 0; y < years; y++ {
		year := y + 1

		// 変動金利: スケジュールで金利が変化したら返済額を残高・残期間で再計算（元利均等のみ）
		if year > 1 && len(input.RateAdjustmentSchedule) > 0 {
			newRate := resolveRateForYear(input.AnnualLoanRate, input.LoanRateDelta, input.RateAdjustmentSchedule, year)
			if newRate != currentRate && remainingBalance > 0 && year <= input.LoanYears {
				if input.LoanMethod != LoanMethodEqualPrincipal {
					remainingYears := input.LoanYears - y
					monthlyPayment = calcMonthlyPayment(remainingBalance, newRate, remainingYears)
				}
				currentRate = newRate
			}
		}

		annualInterest := 0.0
		annualPrincipal := 0.0
		annualLoanPayment := 0.0

		if remainingBalance > 0 && year <= input.LoanYears {
			if input.LoanMethod == LoanMethodEqualPrincipal {
				annualInterest, annualPrincipal = calcYearlyLoanComponentsEqualPrincipal(
					remainingBalance, currentRate, lp.monthlyPrincipalFixed,
				)
				annualLoanPayment = annualInterest + annualPrincipal
			} else {
				annualInterest, annualPrincipal = calcYearlyLoanComponents(
					remainingBalance, currentRate, monthlyPayment,
				)
				annualLoanPayment = monthlyPayment * 12
			}
			remainingBalance -= annualPrincipal
			if remainingBalance < 0 {
				remainingBalance = 0
			}
		}

		// 保有期間内の利息のみ集計（HoldingYears 以内の年）
		if year <= input.HoldingYears {
			totalInterest += annualInterest
		}

		yearAnnualRent := rentForYear(yp.annualRent, input.RentDeclineRate, input.RentGrowthRate, input.RentGrowthYears, y)
		// y は 0-indexed のため float64(y) で 1年目は乗数 1.0 になる（スタート時点の経費率を維持）
		yearExpenseRate := calcEffectiveExpenseRate(input) * math.Pow(1+input.ExpenseInflationRate, float64(y))
		yearExpenses := yearAnnualRent*yearExpenseRate + input.AnnualPropertyTax

		// 減価償却（定額法または定率法）
		var yearDepreciation float64
		switch input.DepreciationMethod {
		case DepreciationMethodDecliningBalance:
			if bookValue > 1.0 {
				yearDepreciation = bookValue * dp.decliningRate
				if bookValue-yearDepreciation < 1.0 {
					yearDepreciation = bookValue - 1.0
				}
				bookValue -= yearDepreciation
			}
		default:
			if year <= dp.usefulLife {
				yearDepreciation = dp.annualDepreciation
			}
		}
		accumulatedDepreciation += yearDepreciation

		// 課税所得 = 収入 - 利息 - 減価償却 - 経費
		taxableIncome := yearAnnualRent - annualInterest - yearDepreciation - yearExpenses
		// 損益通算: 不動産所得が赤字の場合、給与所得との通算により税還付が発生する（負値）
		// 所得税法69条に基づき、負の課税所得は負の incomeTax（税還付）として扱う
		incomeTax := taxableIncome * input.IncomeTaxRate

		capex := capexForYear(input.CapexSchedule, year)
		// 年間ターンオーバーコスト（入退去に伴う原状回復費・AD・フリーレント損失）
		// yearAnnualRent はすでに空室率を加味した実効年間賃料のため、12で割るだけで月額賃料を得る
		monthlyRentForYear := yearAnnualRent / 12
		annualTurnoverCost := calcAnnualTurnoverCost(input, monthlyRentForYear)
		cashFlow := yearAnnualRent - annualLoanPayment - yearExpenses - capex - annualTurnoverCost
		afterTaxCF := cashFlow - incomeTax
		cumulativeCF += afterTaxCF

		// デッドクロス判定: 元金返済額 > 減価償却費 となるゾーン
		// 耐用年数経過後は減価償却=0のため、元金返済が残っていれば即デッドクロス
		// ローン完済後（annualPrincipal==0）はデッドクロスゾーンから脱出
		// 建物費用=0の場合は減価償却対象資産がなくデッドクロスの概念が適用されないため除外
		inDeadCrossZone := input.BuildingCost > 0 && annualPrincipal > 0 && annualPrincipal > yearDepreciation
		isDeadCrossYear := false
		if deadCrossYear == -1 && inDeadCrossZone {
			deadCrossYear = year
			isDeadCrossYear = true
		}

		yearlyResults[y] = YearlyResult{
			Year:                 year,
			CapexAmount:          capex,
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
			EffectiveRate:        currentRate,
		}
	}

	return simulationResult{
		yearlyResults:           yearlyResults,
		accumulatedDepreciation: accumulatedDepreciation,
		deadCrossYear:           deadCrossYear,
		totalInterest:           totalInterest,
	}
}

// calcAllStressScenarios は 6 つのデフォルトシナリオ（入力値が非ゼロなら第 7 カスタムシナリオも）を
// goroutine で並列計算して返す。インデックス固定の slice に直書きするため mutex 不要。
func calcAllStressScenarios(ctx context.Context, input InvestmentInput) []StressScenarioResult {
	type scenarioDef struct {
		label     string
		rateDelta float64
		vacDelta  float64
	}
	allScenarioDefs := [7]scenarioDef{
		{"ベースライン", 0, 0},
		{"金利+1%", 0.01, 0},
		{"金利+2%", 0.02, 0},
		{"空室+10%", 0, 0.10},
		{"空室+20%", 0, 0.20},
		{"複合ストレス", 0.02, 0.10},
		{"カスタム", input.LoanRateDelta, input.VacancyRateDelta},
	}
	hasCustom := input.LoanRateDelta != 0 || input.VacancyRateDelta != 0
	scenarioCount := 6
	if hasCustom {
		scenarioCount = 7
	}
	scenarioResults := make([]StressScenarioResult, scenarioCount)
	var wg sync.WaitGroup
	for i := 0; i < scenarioCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sc := allScenarioDefs[idx]
			scenarioResults[idx] = calcStressScenario(ctx, input, sc.label, sc.rateDelta, sc.vacDelta)
		}(i)
	}
	wg.Wait()
	return scenarioResults
}

// calcStressScenario は指定の金利・空室率オフセットでシナリオ計算を行い結果を返す。
// RateAdjustmentSchedule が設定されている場合、rateDelta はスケジュール金利にも
// 上乗せされる（resolveRateForYear の仕様による）。
// この挙動はフロントエンドの注釈で明示されている。
func calcStressScenario(ctx context.Context, base InvestmentInput, label string, rateDelta, vacDelta float64) StressScenarioResult {
	in := base

	effectiveVacancy := in.VacancyRate + vacDelta
	if effectiveVacancy > 1 {
		effectiveVacancy = 1
	}

	// 初年度賃料（空室率調整済み）— 賃料下落はループ内で年次適用
	annualRent := in.MonthlyRent * 12 * (1 - effectiveVacancy)

	// 初年度の実効金利（スケジュール+ストレスdelta）
	initRate := resolveRateForYear(in.AnnualLoanRate, rateDelta, in.RateAdjustmentSchedule, 1)
	var monthlyPayment float64
	var monthlyPrincipalStress float64
	if in.LoanMethod == LoanMethodEqualPrincipal && in.LoanYears > 0 {
		totalMonths := in.LoanYears * 12
		monthlyPrincipalStress = in.LoanAmount / float64(totalMonths)
	} else {
		monthlyPayment = calcMonthlyPayment(in.LoanAmount, initRate, in.LoanYears)
	}

	// HoldingYears年間の累積CF（税引後）とブレークイーン年を算出
	// DSCR は各年返済額から算出した保有期間内最悪値（変動金利上昇ケースで正確なリスク評価を行うため）
	holdingYears := in.HoldingYears
	if holdingYears <= 0 {
		holdingYears = 10
	}
	totalCF := 0.0
	breakEvenYear := -1
	cumCF := 0.0
	remainingBalance := in.LoanAmount
	currentRate := initRate
	curMonthlyPayment := monthlyPayment
	minDSCR := math.MaxFloat64
	hasLoanYear := false
	for y := 1; y <= holdingYears; y++ {
		// 変動金利スケジュール適用（元利均等のみ月次返済額を再計算）
		if y > 1 && len(in.RateAdjustmentSchedule) > 0 {
			newRate := resolveRateForYear(in.AnnualLoanRate, rateDelta, in.RateAdjustmentSchedule, y)
			if newRate != currentRate && remainingBalance > 0 && y <= in.LoanYears {
				if in.LoanMethod != LoanMethodEqualPrincipal {
					remainingYears := in.LoanYears - (y - 1)
					curMonthlyPayment = calcMonthlyPayment(remainingBalance, newRate, remainingYears)
				}
				currentRate = newRate
			}
		}

		yearLoan := 0.0
		yearInterest := 0.0
		if remainingBalance > 0 && y <= in.LoanYears {
			if in.LoanMethod == LoanMethodEqualPrincipal {
				yi, yp := calcYearlyLoanComponentsEqualPrincipal(remainingBalance, currentRate, monthlyPrincipalStress)
				yearLoan = yi + yp
				yearInterest = yi
				remainingBalance -= yp
			} else {
				annInterest, annPrincipal := calcYearlyLoanComponents(remainingBalance, currentRate, curMonthlyPayment)
				yearLoan = curMonthlyPayment * 12
				yearInterest = annInterest
				remainingBalance -= annPrincipal
			}
			if remainingBalance < 0 {
				remainingBalance = 0
			}
		}

		// 賃料下落率を年次適用（ストレステストは保守的評価のため RentGrowthRate を意図的に無視）
		// RentGrowthRate を考慮した楽観シナリオはメインの Analyze() で rentForYear() が担う。
		declineFactor := math.Pow(1-in.RentDeclineRate, float64(y-1))
		yearRent := annualRent * declineFactor
		// y は 1-indexed のため float64(y-1) で 1年目は乗数 1.0 になる（simulateYears と同一の挙動）
		stressExpenseRate := calcEffectiveExpenseRate(in) * math.Pow(1+in.ExpenseInflationRate, float64(y-1))
		yearExpenses := yearRent*stressExpenseRate + in.AnnualPropertyTax
		yearNOI := yearRent - yearExpenses

		if yearLoan > 0 {
			hasLoanYear = true
			if yearDSCR := yearNOI / yearLoan; yearDSCR < minDSCR {
				minDSCR = yearDSCR
			}
		}

		// 年間ターンオーバーコスト（simulateYears と同様のロジックで算出）
		monthlyRentForYear := yearRent / 12
		annualTurnoverCost := calcAnnualTurnoverCost(in, monthlyRentForYear)

		cf := yearNOI - yearLoan - annualTurnoverCost
		// 減価償却は省略した保守的近似（簡略ストレス計算のため過大に税を見積もる）
		taxableIncome := yearNOI - yearInterest
		incomeTax := 0.0
		if taxableIncome > 0 {
			incomeTax = taxableIncome * in.IncomeTaxRate
		}
		afterTaxCF := cf - incomeTax
		totalCF += afterTaxCF
		cumCF += afterTaxCF
		if breakEvenYear == -1 && cumCF > 0 {
			breakEvenYear = y
		}
	}
	dscr := 0.0
	if hasLoanYear {
		dscr = minDSCR
	}

	// BreakEvenYear は税引後 cumCF が初めて正転した年（#312）
	// IsSafe の DSCR 閾値は dscrSafeThreshold（1.2）でフロントエンド表示と統一（#509）
	isSafe := false
	if !hasLoanYear {
		// 保有期間内に返済が発生しない場合（無借金物件等）はブレークイーン達成のみで安全と判定
		isSafe = breakEvenYear != -1 && breakEvenYear <= holdingYears
	} else {
		isSafe = dscr >= dscrSafeThreshold && breakEvenYear != -1 && breakEvenYear <= holdingYears
	}

	return StressScenarioResult{
		Label:             label,
		InterestRateDelta: rateDelta,
		VacancyRateDelta:  vacDelta,
		TotalCashFlow:     totalCF,
		DSCR:              dscr,
		BreakEvenYear:     breakEvenYear,
		IsSafe:            isSafe,
	}
}

// calcYieldScenarios は楽観・標準・悲観の3シナリオにおける年間実効賃料と表面利回りを算出する。
// 楽観: vacancyRate × 0.5、標準: vacancyRate × 1.0、悲観: vacancyRate × 1.5
// 注意: 表面利回りは満室想定年収/総投資額（空室率に依存しない）であるが、
//
//	AnnualRent（実効賃料）は空室率を反映した値を返す。
func calcYieldScenarios(input InvestmentInput, totalInvestment float64) YieldScenarios {
	grossYield := 0.0
	if totalInvestment > 0 {
		grossYield = (input.MonthlyRent * 12) / totalInvestment
	}

	calcScenario := func(vacancyMultiplier float64) YieldScenario {
		effectiveVacancy := math.Min(input.VacancyRate*vacancyMultiplier, 0.99)
		annualRent := input.MonthlyRent * 12 * (1 - effectiveVacancy)
		return YieldScenario{
			AnnualRent: annualRent,
			GrossYield: grossYield,
		}
	}

	return YieldScenarios{
		Optimistic:  calcScenario(0.5),
		Standard:    calcScenario(1.0),
		Pessimistic: calcScenario(1.5),
	}
}
