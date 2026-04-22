package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	// SqmPerTsubo は 1坪あたりの平方メートル数（mlit パッケージからも参照）
	SqmPerTsubo = 3.30578

	LoanMethodEqualPayment   = "equal-payment"   // 元利均等返済
	LoanMethodEqualPrincipal = "equal-principal"  // 元金均等返済
)

// Analyze は投資入力値から収支シミュレーション結果を算出する
func Analyze(input InvestmentInput) InvestmentResult {
	input.Defaults()

	// ストレステスト値を適用（空室率は99%上限でキャップ）
	effectiveVacancy := math.Min(input.VacancyRate+input.VacancyRateDelta, 0.99)
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

	// 目標利回り逆算
	requiredRent, landDrop := calcRequiredForTarget(input, totalInvestment)

	// ローン月次計算（返済方式に応じて切り替え）
	var monthlyPayment float64
	var monthlyPrincipalFixed float64
	if input.LoanMethod == LoanMethodEqualPrincipal && input.LoanYears > 0 {
		monthlyPrincipalFixed = input.LoanAmount / float64(input.LoanYears*12)
	} else {
		monthlyPayment = calcMonthlyPayment(input.LoanAmount, effectiveRate, input.LoanYears)
	}

	// 減価償却 (定額法)
	// 中古物件は簡便法耐用年数を使用（新築は法定耐用年数）
	usefulLife := CalcResidualUsefulLife(input.BuildingType, input.BuildingAge)
	annualDepreciation := input.BuildingCost / float64(usefulLife)

	// 年次シミュレーション期間の決定
	// max(LoanYears, HoldingYears, 35) を採用する理由:
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
			if input.LoanMethod == LoanMethodEqualPrincipal {
				annualInterest, annualPrincipal = calcYearlyLoanComponentsEqualPrincipal(
					remainingBalance, effectiveRate, monthlyPrincipalFixed,
				)
				annualLoanPayment = annualInterest + annualPrincipal
			} else {
				annualInterest, annualPrincipal = calcYearlyLoanComponents(
					remainingBalance, effectiveRate, monthlyPayment,
				)
				annualLoanPayment = monthlyPayment * 12
			}
			remainingBalance -= annualPrincipal
			if remainingBalance < 0 {
				remainingBalance = 0
			}
		}

		declineFactor := math.Pow(1-input.RentDeclineRate, float64(y))
		yearAnnualRent := annualRent * declineFactor
		yearExpenses := yearAnnualRent*input.ExpenseRate + input.AnnualPropertyTax

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

	// DSCR: 1年目の NOI / 年間返済額
	dscr := 0.0
	if len(yearlyResults) > 0 {
		noi := yearlyResults[0].AnnualRent - yearlyResults[0].AnnualExpenses
		dscr = CalcDSCR(noi, yearlyResults[0].AnnualLoanPayment)
	}

	// 出口戦略 (holdingYears 年後に売却)
	exitSalePrice, exitCapGain, exitTax, exitNet, exitEquity := calcExit(
		input, yearlyResults, accumulatedDepreciation, miscExpenses,
	)

	criticalErrors := calcCriticalErrors(input, deadCrossYear, usefulLife)

	// ストレスシナリオ自動計算（6つのデフォルト + 入力値が非ゼロなら第7シナリオ）
	defaultScenarios := []struct {
		label    string
		rateDelta float64
		vacDelta  float64
	}{
		{"ベースライン", 0, 0},
		{"金利+1%", 0.01, 0},
		{"金利+2%", 0.02, 0},
		{"空室+10%", 0, 0.10},
		{"空室+20%", 0, 0.20},
		{"複合ストレス", 0.02, 0.10},
	}
	stressScenarios := make([]StressScenarioResult, 0, 7)
	for _, sc := range defaultScenarios {
		stressScenarios = append(stressScenarios, calcStressScenario(input, sc.label, sc.rateDelta, sc.vacDelta))
	}
	if input.LoanRateDelta != 0 || input.VacancyRateDelta != 0 {
		stressScenarios = append(stressScenarios, calcStressScenario(
			input, "カスタム", input.LoanRateDelta, input.VacancyRateDelta,
		))
	}

	acquisitionCosts := CalcAcquisitionCosts(
		input.LandPrice,
		input.BuildingCost,
		AcquisitionCostOptions{
			BrokerageMultiplier: 1.0,
			LoanAmount:          input.LoanAmount,
		},
	)

	yieldScenarios := calcYieldScenarios(input, totalInvestment)
	ltvSensitivity := CalcLTVSensitivity(input, nil)

	return InvestmentResult{
		TotalInvestment:       totalInvestment,
		MiscExpenses:          miscExpenses,
		GrossYield:            grossYield,
		NetYield:              netYield,
		IsAboveYieldTarget:    grossYield >= input.YieldTarget,
		YieldTarget:           input.YieldTarget,
		RequiredCostReduction: landDrop,
		RequiredMonthlyRent:   requiredRent,
		DeadCrossYear:         deadCrossYear,
		YearlyResults:         yearlyResults,
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
	}
}

// calcYieldScenarios は楽観・標準・悲観の3シナリオにおける年間実効賃料と表面利回りを算出する。
// 楽観: vacancyRate × 0.5、標準: vacancyRate × 1.0、悲観: vacancyRate × 1.5
// 注意: 表面利回りは満室想定年収/総投資額（空室率に依存しない）であるが、
//   AnnualRent（実効賃料）は空室率を反映した値を返す。
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

// calcStressScenario は指定の金利・空室率オフセットでシナリオ計算を行い結果を返す
func calcStressScenario(base InvestmentInput, label string, rateDelta, vacDelta float64) StressScenarioResult {
	in := base

	effectiveVacancy := in.VacancyRate + vacDelta
	if effectiveVacancy > 1 {
		effectiveVacancy = 1
	}
	effectiveRate := in.AnnualLoanRate + rateDelta

	annualRent := in.MonthlyRent * 12 * (1 - effectiveVacancy)
	annualExpenses := annualRent*in.ExpenseRate + in.AnnualPropertyTax
	noi := annualRent - annualExpenses

	var monthlyPayment float64
	var monthlyPrincipalStress float64
	var annualLoanPayment float64
	if in.LoanMethod == LoanMethodEqualPrincipal && in.LoanYears > 0 {
		totalMonths := in.LoanYears * 12
		monthlyPrincipalStress = in.LoanAmount / float64(totalMonths)
		yi, yp := calcYearlyLoanComponentsEqualPrincipal(in.LoanAmount, effectiveRate, monthlyPrincipalStress)
		annualLoanPayment = yi + yp
	} else {
		monthlyPayment = calcMonthlyPayment(in.LoanAmount, effectiveRate, in.LoanYears)
		annualLoanPayment = monthlyPayment * 12
	}

	// DSCR = NOI / 年間ローン返済額
	dscr := 0.0
	if annualLoanPayment > 0 {
		dscr = noi / annualLoanPayment
	}

	// HoldingYears年間の累積CF（税引前）とブレークイーン年を算出
	holdingYears := in.HoldingYears
	if holdingYears <= 0 {
		holdingYears = 10
	}
	totalCF := 0.0
	breakEvenYear := -1
	cumCF := 0.0
	remainingBalance := in.LoanAmount
	for y := 1; y <= holdingYears; y++ {
		yearLoan := annualLoanPayment
		if remainingBalance <= 0 || y > in.LoanYears {
			yearLoan = 0
		}
		cf := annualRent - yearLoan - annualExpenses
		totalCF += cf
		cumCF += cf
		if breakEvenYear == -1 && cumCF > 0 {
			breakEvenYear = y
		}
		// 残高更新
		if remainingBalance > 0 && y <= in.LoanYears {
			var annPrincipal float64
			if in.LoanMethod == LoanMethodEqualPrincipal {
				_, annPrincipal = calcYearlyLoanComponentsEqualPrincipal(remainingBalance, effectiveRate, monthlyPrincipalStress)
			} else {
				_, annPrincipal = calcYearlyLoanComponents(remainingBalance, effectiveRate, monthlyPayment)
			}
			remainingBalance -= annPrincipal
			if remainingBalance < 0 {
				remainingBalance = 0
			}
		}
	}

	isSafe := false
	if annualLoanPayment == 0 {
		// 無借金物件はDSCRによる返済リスクがないため、ブレークイーン達成のみで安全と判定
		isSafe = breakEvenYear != -1 && breakEvenYear <= holdingYears
	} else {
		isSafe = dscr >= 1.0 && breakEvenYear != -1 && breakEvenYear <= holdingYears
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

// CalcDSCR は借入金償還余裕率を返す（DSCR = NOI / 年間返済額）
// annualDebtService が 0 以下の場合は 0 を返す。
func CalcDSCR(noi, annualDebtService float64) float64 {
	if annualDebtService <= 0 {
		return 0
	}
	return noi / annualDebtService
}

// CalcEqualPrincipalPayment は元金均等返済の month 回目（1始まり）の月次返済額を返す。
// principal: 借入元本, annualRate: 年利, months: 総返済回数, month: 対象月（1〜months）
func CalcEqualPrincipalPayment(principal, annualRate float64, months, month int) float64 {
	if principal <= 0 || months <= 0 || month < 1 || month > months {
		return 0
	}
	monthlyPrincipal := principal / float64(months)
	remainingBalance := principal - float64(month-1)*monthlyPrincipal
	monthlyInterest := remainingBalance * (annualRate / 12)
	return monthlyPrincipal + monthlyInterest
}

// calcYearlyLoanComponentsEqualPrincipal は元金均等返済における1年分の利息・元金返済額を計算する。
// monthlyPrincipal: 毎月の固定元金返済額（= 元本 / 総返済回数）
func calcYearlyLoanComponentsEqualPrincipal(balance, annualRate, monthlyPrincipal float64) (interest, principal float64) {
	r := annualRate / 12
	remaining := balance
	for range 12 {
		if remaining <= 0 {
			break
		}
		mp := monthlyPrincipal
		if mp > remaining {
			mp = remaining
		}
		interest += remaining * r
		principal += mp
		remaining -= mp
	}
	return interest, principal
}

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

	rows := make([]LTVSensitivityRow, 0, len(ltvRange))
	for _, ltv := range ltvRange {
		loanAmount := totalInvestment * ltv
		equity := totalInvestment * (1 - ltv)

		var annualDebtService float64
		if input.LoanMethod == LoanMethodEqualPrincipal && input.LoanYears > 0 {
			totalMonths := input.LoanYears * 12
			monthlyPrincipal := loanAmount / float64(totalMonths)
			yearInterest, yearPrincipal := calcYearlyLoanComponentsEqualPrincipal(
				loanAmount, input.AnnualLoanRate, monthlyPrincipal,
			)
			annualDebtService = yearInterest + yearPrincipal
		} else {
			mp := calcMonthlyPayment(loanAmount, input.AnnualLoanRate, input.LoanYears)
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

// 譲渡所得税率（所得税＋復興特別所得税＋住民税）
// 根拠: 租税特別措置法31条・32条、復興財源確保法33条（2037年まで）
// 注意: 租税特別措置法31条の3の10年超軽減（14.21%）は「居住用財産」の特例であり
//       投資用物件には適用されない。投資用は保有年数に関わらず長期=20.315%。
const (
	shortTermTransferTaxRate = 0.3963  // 短期（5年以下）: 所得税30%+復興0.63%+住民税9%
	longTermTransferTaxRate  = 0.20315 // 長期（5年超）: 所得税15%+復興0.315%+住民税5%
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
		// 投資用物件の譲渡所得税: 保有5年超=長期(20.315%)、5年以下=短期(39.63%)の2段階
		// 租税特別措置法31条の3の10年超軽減(14.21%)は居住用財産の特例のため対象外
		var taxRate float64
		if input.HoldingYears > 5 {
			taxRate = longTermTransferTaxRate
		} else {
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

// calcZoningSummary は取引データから最頻の用途地域情報を抽出する
func calcZoningSummary(transactions []LandTransaction) *ZoningSummary {
	if len(transactions) == 0 {
		return nil
	}
	cp := modalString(transactions, func(t LandTransaction) string { return t.CityPlanning })
	bc := modalString(transactions, func(t LandTransaction) string { return t.BuildingCoverage })
	far := modalString(transactions, func(t LandTransaction) string { return t.FloorAreaRatio })
	if cp == "" && bc == "" && far == "" {
		return nil
	}
	return &ZoningSummary{
		CityPlanning:     cp,
		BuildingCoverage: bc,
		FloorAreaRatio:   far,
	}
}

// detectUrbanRisks は取引データと用途地域サマリーから都市計画リスクを検出する
func detectUrbanRisks(transactions []LandTransaction, zoning *ZoningSummary) []UrbanRisk {
	var risks []UrbanRisk
	if zoning == nil {
		return risks
	}

	// 市街化調整区域（最頻値が調整区域）
	if strings.Contains(zoning.CityPlanning, "市街化調整区域") {
		risks = append(risks, UrbanRisk{
			Code:        "URBANIZATION_CONTROL_ZONE",
			Level:       UrbanRiskLevelError,
			Title:       "市街化調整区域",
			Description: "原則として新たな建築・用途変更が制限されます。既存建物の建替えが困難になる可能性があります。",
		})
	}

	// 都市計画区域外 / 非線引き区域
	if strings.Contains(zoning.CityPlanning, "非線引") || zoning.CityPlanning == "都市計画区域外" {
		risks = append(risks, UrbanRisk{
			Code:        "UNZONED_AREA",
			Level:       UrbanRiskLevelWarning,
			Title:       "非線引き・都市計画区域外",
			Description: "インフラ整備が遅れやすく、将来の資産価値が不安定になる可能性があります。",
		})
	}

	// 調整区域が最頻値でなくとも30%以上混在する場合の注意
	if !strings.Contains(zoning.CityPlanning, "市街化調整区域") {
		controlCount, totalWithCP := 0, 0
		for _, t := range transactions {
			if t.CityPlanning != "" {
				totalWithCP++
				if strings.Contains(t.CityPlanning, "市街化調整区域") {
					controlCount++
				}
			}
		}
		if totalWithCP > 0 && float64(controlCount)/float64(totalWithCP) >= 0.3 {
			ratio := float64(controlCount) / float64(totalWithCP) * 100
			risks = append(risks, UrbanRisk{
				Code:        "MIXED_ZONE_CAUTION",
				Level:       UrbanRiskLevelWarning,
				Title:       "市街化調整区域が混在",
				Description: fmt.Sprintf("エリア内取引の約%.0f%%が市街化調整区域です。対象物件の区域区分を必ず確認してください。", ratio),
			})
		}
	}

	return risks
}

// BuildUrbanRisksFromAPIs は MLIT 専用 API（XKT003/020/030/XST001）の結果からリスクを構築する。
// detectUrbanRisks（XIT001テキスト検出）とは独立して使用し、結果を呼び出し元でマージする。
func BuildUrbanRisksFromAPIs(
	locationItems []LocationOptimizationItem,
	embankmentItems []EmbankmentItem,
	roadItems []UrbanRoadItem,
	disasters []DisasterHistoryItem,
) []UrbanRisk {
	var risks []UrbanRisk

	// XKT003: 立地適正化計画フィーチャが存在し、居住誘導区域が含まれない場合
	if len(locationItems) > 0 {
		hasResidential := false
		for _, item := range locationItems {
			if strings.Contains(item.KubunNameJa, "居住誘導区域") {
				hasResidential = true
				break
			}
		}
		if !hasResidential {
			risks = append(risks, UrbanRisk{
				Code:        "OUTSIDE_RESIDENTIAL_GUIDANCE",
				Level:       UrbanRiskLevelWarning,
				Title:       "居住誘導区域外",
				Description: "立地適正化計画の居住誘導区域外です。将来的に行政サービスの縮小・インフラ維持コスト増加の可能性があります（コンパクトシティ計画）。",
			})
		}
	}

	// XKT020: 大規模盛土造成地フィーチャが存在する場合
	if len(embankmentItems) > 0 {
		embDesc := "大規模盛土造成地に該当します。地震時の沈下・崩壊リスクがあります。"
		if c := embankmentItems[0].Classification; c != "" {
			embDesc = fmt.Sprintf("大規模盛土造成地（%s）に該当します。地震時の沈下・崩壊リスクがあります。", c)
		}
		risks = append(risks, UrbanRisk{
			Code:        "LARGE_EMBANKMENT",
			Level:       UrbanRiskLevelWarning,
			Title:       "大規模盛土造成地",
			Description: embDesc,
		})
	}

	// XKT030: 都市計画道路（kubun_id=3011）フィーチャが存在する場合
	for _, item := range roadItems {
		if item.KubunID == 3011 {
			risks = append(risks, UrbanRisk{
				Code:        "URBAN_PLANNING_ROAD",
				Level:       UrbanRiskLevelWarning,
				Title:       "都市計画道路の予定地",
				Description: "都市計画道路の予定地に一部かかっています。将来的に建物の一部または全部が収用対象となる可能性があります。",
			})
			break
		}
	}

	// XST001: 災害履歴フィーチャが存在する場合
	if len(disasters) > 0 {
		names := make([]string, 0, len(disasters))
		seen := make(map[string]bool)
		for _, d := range disasters {
			if d.Name != "" && !seen[d.Name] {
				names = append(names, d.Name)
				seen[d.Name] = true
			}
		}
		desc := "このエリアで過去に災害が記録されています。"
		if len(names) > 0 {
			desc = fmt.Sprintf("このエリアで過去に災害が記録されています（%s）。", strings.Join(names, "・"))
		}
		risks = append(risks, UrbanRisk{
			Code:        "DISASTER_HISTORY",
			Level:       UrbanRiskLevelWarning,
			Title:       "災害履歴あり",
			Description: desc,
		})
	}

	return risks
}

// modalString は transactions から getter で取得した文字列の最頻値を返す（空文字は除外）
func modalString(transactions []LandTransaction, getter func(LandTransaction) string) string {
	counts := make(map[string]int, len(transactions))
	for _, t := range transactions {
		v := getter(t)
		if v != "" {
			counts[v]++
		}
	}
	best, bestCount := "", 0
	for v, c := range counts {
		if c > bestCount {
			best, bestCount = v, c
		}
	}
	return best
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
